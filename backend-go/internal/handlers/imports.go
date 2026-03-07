package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/anthropics/accounting-tool/backend-go/internal/config"
	"github.com/anthropics/accounting-tool/backend-go/internal/models"
	"github.com/anthropics/accounting-tool/backend-go/internal/services"
)

type ImportHandler struct {
	db     *sqlx.DB
	cfg    *config.Config
	gemini *services.GeminiExtractService
}

func NewImportHandler(db *sqlx.DB, cfg *config.Config, gemini *services.GeminiExtractService) *ImportHandler {
	return &ImportHandler{db: db, cfg: cfg, gemini: gemini}
}

func (h *ImportHandler) Routes(r chi.Router) {
	r.Post("/api/imports/upload", h.Upload)
	r.Get("/api/imports", h.List)
	r.Get("/api/imports/{sessionId}", h.GetSession)
	r.Patch("/api/imports/{sessionId}/items/{itemId}", h.UpdateItem)
	r.Post("/api/imports/{sessionId}/confirm", h.Confirm)
	r.Get("/api/imports/{sessionId}/source", h.GetSource)
	r.Get("/api/imports/{sessionId}/pages/{pageNum}", h.GetPage)
	r.Delete("/api/imports/{sessionId}", h.DeleteSession)
}

func (h *ImportHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "No file provided")
		return
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read uploaded file")
		return
	}
	filename := header.Filename
	lower := strings.ToLower(filename)

	if !strings.HasSuffix(lower, ".pdf") {
		if strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg") ||
			strings.HasSuffix(lower, ".png") || strings.HasSuffix(lower, ".webp") {
			writeJSON(w, http.StatusOK, models.ImportUploadResult{
				Success: false,
				Message: "Receipt images should be uploaded on the Receipts page.",
				Errors:  []string{"Use the Receipts page for receipt images"},
			})
			return
		}
		writeJSON(w, http.StatusOK, models.ImportUploadResult{
			Success: false,
			Message: "Unsupported file type. Upload a bank statement PDF.",
			Errors:  []string{"Unsupported file type"},
		})
		return
	}

	fileHash := services.ComputeFileHash(content)

	// Check existing pending session
	var existingID string
	err = h.db.Get(&existingID,
		`SELECT id FROM import_sessions WHERE file_hash = ? AND status = 'pending'`, fileHash)
	if err == nil {
		writeJSON(w, http.StatusOK, models.ImportUploadResult{
			Success:   true,
			SessionID: &existingID,
			Message:   "This file has already been uploaded and is pending review.",
		})
		return
	}

	// Save file
	savePath := filepath.Join(h.cfg.UploadsDir, filepath.Base(filename))
	if err := os.WriteFile(savePath, content, 0o600); err != nil {
		log.Printf("import upload save: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to save uploaded file")
		return
	}

	// Call Gemini
	ok, data, errMsg, usage := h.gemini.ExtractFromPDF(content)
	if !ok || data == nil {
		errs := []string{errMsg}
		if errMsg == "" {
			errs = []string{"Unknown error"}
		}
		writeJSON(w, http.StatusOK, models.ImportUploadResult{
			Success: false, Message: "AI extraction failed.", Errors: errs,
		})
		return
	}

	sessionID := uuid.New().String()

	metaJSON, _ := json.Marshal(map[string]interface{}{
		"period_start":    data["period_start"],
		"period_end":      data["period_end"],
		"opening_balance": data["opening_balance"],
		"closing_balance": data["closing_balance"],
	})

	var usageJSON *string
	if usage != nil {
		uj, _ := json.Marshal(usage)
		s := string(uj)
		usageJSON = &s
	}

	// We don't have PyMuPDF in Go, so page images are skipped for now.
	// The frontend can use the PDF source endpoint directly.
	if _, err := h.db.Exec(
		`INSERT INTO import_sessions (id, source_type, source_file, file_hash, file_path, page_count, metadata_json, ai_usage_json, status)
		 VALUES (?, 'pdf', ?, ?, ?, ?, ?, ?, 'pending')`,
		sessionID, filename, fileHash, savePath, nil, string(metaJSON), usageJSON); err != nil {
		log.Printf("import session insert: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Create import items
	txns, _ := data["transactions"].([]interface{})
	for _, raw := range txns {
		txnMap, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		dateStr := services.ParseDate(strOrDefault(txnMap["date"]))
		desc, _ := txnMap["description"].(string)
		amount, _ := txnMap["amount"].(float64)
		balance, _ := txnMap["balance"].(float64)
		pageNum, _ := txnMap["page_number"].(float64)
		rawJSON, _ := json.Marshal(txnMap)

		var balVal interface{} = nil
		if balance != 0 {
			balVal = balance
		}
		var pageVal interface{} = nil
		if pageNum != 0 {
			pageVal = int(pageNum)
		}
		var dateVal interface{} = nil
		if dateStr != "" {
			dateVal = dateStr
		}

		if _, err := h.db.Exec(
			`INSERT INTO import_items (session_id, page_num, extracted_date, extracted_description, extracted_amount, extracted_balance, raw_ai_json, status)
			 VALUES (?, ?, ?, ?, ?, ?, ?, 'pending')`,
			sessionID, pageVal, dateVal, desc, amount, balVal, string(rawJSON)); err != nil {
			log.Printf("import item insert: %v", err)
		}
	}

	// Duplicate detection
	type itemForDup struct {
		ID     int     `db:"id"`
		Date   string  `db:"extracted_date"`
		Amount float64 `db:"extracted_amount"`
		Desc   string  `db:"extracted_description"`
	}
	var items []itemForDup
	if err := h.db.Select(&items, `SELECT id, COALESCE(extracted_date,'') as extracted_date, COALESCE(extracted_amount,0) as extracted_amount, COALESCE(extracted_description,'') as extracted_description FROM import_items WHERE session_id = ?`, sessionID); err != nil {
		log.Printf("import items select: %v", err)
	}

	for _, item := range items {
		dupItems := []map[string]interface{}{
			{"date": item.Date, "amount": item.Amount, "description": item.Desc},
		}
		services.DetectDuplicates(h.db, dupItems)
		di := dupItems[0]
		if dupID, ok := di["duplicate_of_id"].(int); ok {
			score, _ := di["duplicate_score"].(float64)
			reason, _ := di["duplicate_reason"].(string)
			if _, err := h.db.Exec(`UPDATE import_items SET duplicate_of_id = ?, duplicate_score = ?, duplicate_reason = ? WHERE id = ?`,
				dupID, score, reason, item.ID); err != nil {
				log.Printf("import item duplicate update: %v", err)
			}
		}
	}

	sid := sessionID
	writeJSON(w, http.StatusOK, models.ImportUploadResult{
		Success:   true,
		SessionID: &sid,
		Message:   fmt.Sprintf("Extracted %d transactions.", len(txns)),
	})
}

func (h *ImportHandler) List(w http.ResponseWriter, r *http.Request) {
	var sessions []models.ImportSession
	h.db.Select(&sessions, `SELECT * FROM import_sessions ORDER BY created_at DESC`)

	resp := make([]models.ImportSessionResponse, 0, len(sessions))
	for _, s := range sessions {
		resp = append(resp, h.sessionToResponse(s))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *ImportHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	var session models.ImportSession
	err := h.db.Get(&session, `SELECT * FROM import_sessions WHERE id = ?`, sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Import session not found")
		return
	}
	writeJSON(w, http.StatusOK, h.sessionToResponse(session))
}

func (h *ImportHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	itemID, err := strconv.Atoi(chi.URLParam(r, "itemId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	var count int
	h.db.Get(&count, `SELECT COUNT(*) FROM import_items WHERE id = ? AND session_id = ?`, itemID, sessionID)
	if count == 0 {
		writeError(w, http.StatusNotFound, "Import item not found")
		return
	}

	var req models.ImportItemUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	var sets []string
	var args []interface{}
	if req.ExtractedDate != nil {
		sets = append(sets, "extracted_date = ?")
		args = append(args, *req.ExtractedDate)
	}
	if req.ExtractedDescription != nil {
		sets = append(sets, "extracted_description = ?")
		args = append(args, *req.ExtractedDescription)
	}
	if req.ExtractedAmount != nil {
		sets = append(sets, "extracted_amount = ?")
		args = append(args, *req.ExtractedAmount)
	}
	if req.ExtractedBalance != nil {
		sets = append(sets, "extracted_balance = ?")
		args = append(args, *req.ExtractedBalance)
	}
	if req.ExtractedMerchant != nil {
		sets = append(sets, "extracted_merchant = ?")
		args = append(args, *req.ExtractedMerchant)
	}
	if req.Status != nil {
		sets = append(sets, "status = ?")
		args = append(args, *req.Status)
	}
	if req.DuplicateOfID != nil {
		sets = append(sets, "duplicate_of_id = ?")
		args = append(args, *req.DuplicateOfID)
	}

	if len(sets) > 0 {
		args = append(args, itemID)
		if _, err := h.db.Exec("UPDATE import_items SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...); err != nil {
			log.Printf("import item update: %v", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
	}

	var item models.ImportItem
	h.db.Get(&item, `SELECT * FROM import_items WHERE id = ?`, itemID)
	writeJSON(w, http.StatusOK, h.itemToResponse(item))
}

func (h *ImportHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")

	var session models.ImportSession
	err := h.db.Get(&session, `SELECT * FROM import_sessions WHERE id = ?`, sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Import session not found")
		return
	}
	if session.Status != "pending" {
		writeError(w, http.StatusBadRequest, "Session already processed")
		return
	}

	importSvc := services.NewImportService(h.db)
	classifySvc := services.NewClassifyService(h.db)
	merchantSvc := services.NewMerchantService(h.db)

	// For PDFs, create a Statement record
	var statementID *int
	if session.SourceType == "pdf" && session.MetadataJSON.Valid {
		var meta map[string]interface{}
		json.Unmarshal([]byte(session.MetadataJSON.String), &meta)

		accountID := importSvc.EnsureAccount("import-primary", "Primary Import Account", "Import", "current", h.cfg.DefaultCurrency)

		periodStart := services.ParseDate(strOrDefault(meta["period_start"]))
		periodEnd := services.ParseDate(strOrDefault(meta["period_end"]))
		if periodStart == "" {
			periodStart = time.Now().Format("2006-01-02")
		}
		if periodEnd == "" {
			periodEnd = time.Now().Format("2006-01-02")
		}

		ps, _ := time.Parse("2006-01-02", periodStart)
		pe, _ := time.Parse("2006-01-02", periodEnd)

		var ob, cb *float64
		if v, ok := meta["opening_balance"].(float64); ok {
			ob = &v
		}
		if v, ok := meta["closing_balance"].(float64); ok {
			cb = &v
		}

		sid, err := importSvc.CreateStatement(accountID, session.SourceFile, session.FileHash, ps, pe, ob, cb, "")
		if err == nil {
			statementID = &sid
		}
	}

	tx, err := h.db.Beginx()
	if err != nil {
		log.Printf("import confirm begin tx: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer tx.Rollback()

	var items []models.ImportItem
	if err := tx.Select(&items, `SELECT * FROM import_items WHERE session_id = ? ORDER BY id`, sessionID); err != nil {
		log.Printf("import confirm select items: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	created := 0
	linked := 0
	skipped := 0

	for _, item := range items {
		if item.Status == "skipped" {
			skipped++
			continue
		}
		if item.DuplicateOfID.Valid && item.Status != "confirmed" {
			skipped++
			continue
		}
		if item.DuplicateOfID.Valid {
			linked++
			continue
		}

		if !item.ExtractedDate.Valid || !item.ExtractedAmount.Valid {
			skipped++
			continue
		}

		desc := ""
		if item.ExtractedDescription.Valid {
			desc = item.ExtractedDescription.String
		}

		itemDate, _ := time.Parse("2006-01-02", item.ExtractedDate.String)
		var bal *float64
		if item.ExtractedBalance.Valid {
			bal = &item.ExtractedBalance.Float64
		}

		sourceHash := services.ComputeTransactionHash(itemDate, desc, item.ExtractedAmount.Float64, bal)

		var dupCount int
		tx.Get(&dupCount, `SELECT COUNT(*) FROM transactions WHERE source_hash = ?`, sourceHash)
		if dupCount > 0 {
			skipped++
			continue
		}

		// Fuzzy match
		if session.SourceType == "receipt_image" {
			fuzzyID := importSvc.FindFuzzyMatch(itemDate, item.ExtractedAmount.Float64, desc, false, true)
			if fuzzyID != nil {
				if _, err := tx.Exec(`UPDATE import_items SET status = 'confirmed', duplicate_of_id = ? WHERE id = ?`, *fuzzyID, item.ID); err != nil {
					log.Printf("import confirm fuzzy update: %v", err)
				}
				linked++
				continue
			}
		} else if session.SourceType == "pdf" && statementID != nil {
			fuzzyID := importSvc.FindFuzzyMatch(itemDate, item.ExtractedAmount.Float64, desc, true, false)
			if fuzzyID != nil {
				var balVal interface{} = nil
				if bal != nil {
					balVal = *bal
				}
				if _, err := tx.Exec(
					`UPDATE transactions SET source_hash = ?, statement_id = ?, raw_description = ?, raw_balance = ?, raw_amount = ? WHERE id = ?`,
					sourceHash, *statementID, desc, balVal, item.ExtractedAmount.Float64, *fuzzyID); err != nil {
					log.Printf("import confirm txn update: %v", err)
				}

				var catSource string
				tx.Get(&catSource, `SELECT category_source FROM transactions WHERE id = ?`, *fuzzyID)
				if catSource != "manual" && catSource != "receipt" {
					catID, src := classifySvc.Classify(desc)
					if catID != nil {
						if _, err := tx.Exec(`UPDATE transactions SET category_id = ?, category_source = ? WHERE id = ?`, *catID, src, *fuzzyID); err != nil {
							log.Printf("import confirm classify update: %v", err)
						}
					}
				}
				if _, err := tx.Exec(`UPDATE import_items SET status = 'confirmed' WHERE id = ?`, item.ID); err != nil {
					log.Printf("import confirm item status: %v", err)
				}
				linked++
				continue
			}
		}

		resolvedDesc := merchantSvc.Resolve(desc)

		var stmtIDVal interface{} = nil
		if statementID != nil {
			stmtIDVal = *statementID
		}
		var balVal interface{} = nil
		if bal != nil {
			balVal = *bal
		}

		catID, catSource := classifySvc.Classify(desc)
		var catIDVal interface{} = nil
		if catID != nil {
			catIDVal = *catID
		}

		if _, err := tx.Exec(
			`INSERT INTO transactions
			 (statement_id, source_hash, raw_date, raw_description, raw_amount, raw_balance, amount, description, category_id, category_source)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			stmtIDVal, sourceHash, item.ExtractedDate.String, desc, item.ExtractedAmount.Float64,
			balVal, item.ExtractedAmount.Float64, resolvedDesc, catIDVal, catSource); err != nil {
			log.Printf("import confirm txn insert: %v", err)
		}

		if resolvedDesc != desc {
			merchantSvc.GetOrCreate(resolvedDesc, desc)
		}

		if item.ExtractedMerchant.Valid && item.ExtractedMerchant.String != "" {
			var rawAIMerchant string
			if item.RawAIJSON.Valid {
				var aiData map[string]interface{}
				json.Unmarshal([]byte(item.RawAIJSON.String), &aiData)
				rawAIMerchant, _ = aiData["merchant_name"].(string)
			}
			merchantSvc.GetOrCreate(item.ExtractedMerchant.String, rawAIMerchant)
		}

		if _, err := tx.Exec(`UPDATE import_items SET status = 'confirmed' WHERE id = ?`, item.ID); err != nil {
			log.Printf("import confirm item status: %v", err)
		}
		created++
	}

	if _, err := tx.Exec(`UPDATE import_sessions SET status = 'confirmed' WHERE id = ?`, sessionID); err != nil {
		log.Printf("import confirm session status: %v", err)
	}

	if err := tx.Commit(); err != nil {
		log.Printf("import confirm commit: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"created": created,
		"linked":  linked,
		"skipped": skipped,
		"message": fmt.Sprintf("Created %d transactions, linked %d, skipped %d.", created, linked, skipped),
	})
}

func (h *ImportHandler) GetSource(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	var filePath string
	var sourceFile string
	err := h.db.QueryRow(`SELECT file_path, source_file FROM import_sessions WHERE id = ?`, sessionID).Scan(&filePath, &sourceFile)
	if err != nil {
		writeError(w, http.StatusNotFound, "Import session not found")
		return
	}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "Source file not found on disk")
		return
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	mediaTypes := map[string]string{
		".pdf": "application/pdf", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
		".png": "image/png", ".webp": "image/webp",
	}
	mt := mediaTypes[ext]
	if mt == "" {
		mt = "application/octet-stream"
	}
	w.Header().Set("Content-Type", mt)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": filepath.Base(sourceFile)}))
	http.ServeFile(w, r, filePath)
}

func (h *ImportHandler) GetPage(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	pageNumStr := chi.URLParam(r, "pageNum")

	// Validate sessionID format (UUID)
	if _, err := uuid.Parse(sessionID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid session ID")
		return
	}
	// Validate pageNum is positive integer
	pageNum, err := strconv.Atoi(pageNumStr)
	if err != nil || pageNum < 1 {
		writeError(w, http.StatusBadRequest, "invalid page number")
		return
	}

	imgPath := filepath.Join(h.cfg.PageImagesDir, sessionID, fmt.Sprintf("page_%d.png", pageNum))
	if _, err := os.Stat(imgPath); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "Page image not found")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	http.ServeFile(w, r, imgPath)
}

func (h *ImportHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")

	var filePath string
	err := h.db.Get(&filePath, `SELECT file_path FROM import_sessions WHERE id = ?`, sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Import session not found")
		return
	}

	os.Remove(filePath)
	os.RemoveAll(filepath.Join(h.cfg.PageImagesDir, sessionID))

	if _, err := h.db.Exec(`DELETE FROM import_items WHERE session_id = ?`, sessionID); err != nil {
		log.Printf("import delete items: %v", err)
	}
	if _, err := h.db.Exec(`DELETE FROM import_sessions WHERE id = ?`, sessionID); err != nil {
		log.Printf("import delete session: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "Import session deleted."})
}

// Helpers

func (h *ImportHandler) sessionToResponse(s models.ImportSession) models.ImportSessionResponse {
	resp := models.ImportSessionResponse{
		ID:         s.ID,
		SourceType: s.SourceType,
		SourceFile: s.SourceFile,
		FileHash:   s.FileHash,
		Status:     s.Status,
		CreatedAt:  s.CreatedAt.Format("2006-01-02T15:04:05"),
	}
	if s.PageCount.Valid {
		v := int(s.PageCount.Int64)
		resp.PageCount = &v
	}
	if s.PageImagePaths.Valid {
		json.Unmarshal([]byte(s.PageImagePaths.String), &resp.PageImagePaths)
	}
	if s.MetadataJSON.Valid {
		resp.MetadataJSON = &s.MetadataJSON.String
	}
	if s.AIUsageJSON.Valid {
		resp.AIUsageJSON = &s.AIUsageJSON.String
	}

	var items []models.ImportItem
	h.db.Select(&items, `SELECT * FROM import_items WHERE session_id = ? ORDER BY id`, s.ID)

	resp.Items = make([]models.ImportItemResponse, 0, len(items))
	for _, item := range items {
		resp.Items = append(resp.Items, h.itemToResponse(item))
	}
	return resp
}

func (h *ImportHandler) itemToResponse(item models.ImportItem) models.ImportItemResponse {
	resp := models.ImportItemResponse{
		ID:        item.ID,
		SessionID: item.SessionID,
		Status:    item.Status,
		CreatedAt: item.CreatedAt.Format("2006-01-02T15:04:05"),
	}
	if item.PageNum.Valid {
		v := int(item.PageNum.Int64)
		resp.PageNum = &v
	}
	if item.ExtractedDate.Valid {
		resp.ExtractedDate = &item.ExtractedDate.String
	}
	if item.ExtractedDescription.Valid {
		resp.ExtractedDescription = &item.ExtractedDescription.String
	}
	if item.ExtractedAmount.Valid {
		resp.ExtractedAmount = &item.ExtractedAmount.Float64
	}
	if item.ExtractedBalance.Valid {
		resp.ExtractedBalance = &item.ExtractedBalance.Float64
	}
	if item.ExtractedMerchant.Valid {
		resp.ExtractedMerchant = &item.ExtractedMerchant.String
	}
	if item.ExtractedItemsJSON.Valid {
		resp.ExtractedItemsJSON = &item.ExtractedItemsJSON.String
	}
	if item.DuplicateOfID.Valid {
		resp.DuplicateOfID = &item.DuplicateOfID.Int64
		// Fetch duplicate transaction info
		type dupTxn struct {
			RawDate        string  `db:"raw_date"`
			RawDescription string  `db:"raw_description"`
			Amount         float64 `db:"amount"`
		}
		var dt dupTxn
		if err := h.db.Get(&dt, `SELECT raw_date, raw_description, amount FROM transactions WHERE id = ?`, item.DuplicateOfID.Int64); err == nil {
			resp.DuplicateTransactionDate = &dt.RawDate
			resp.DuplicateTransactionDescription = &dt.RawDescription
			resp.DuplicateTransactionAmount = &dt.Amount
		}
	}
	if item.DuplicateScore.Valid {
		resp.DuplicateScore = &item.DuplicateScore.Float64
	}
	if item.DuplicateReason.Valid {
		resp.DuplicateReason = &item.DuplicateReason.String
	}
	return resp
}

func strOrDefault(v interface{}) string {
	s, _ := v.(string)
	return s
}

// Unused suppression
var _ = math.Round
