package handlers

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/anthropics/accounting-tool/backend-go/internal/config"
	"github.com/anthropics/accounting-tool/backend-go/internal/models"
	"github.com/anthropics/accounting-tool/backend-go/internal/services"
)

type receiptRow struct {
	ID                   int             `db:"id"`
	ImagePath            string          `db:"image_path"`
	ImageHash            string          `db:"image_hash"`
	MerchantName         sql.NullString  `db:"merchant_name"`
	ReceiptDate          sql.NullString  `db:"receipt_date"`
	ReceiptTime          sql.NullString  `db:"receipt_time"`
	TotalAmount          sql.NullFloat64 `db:"total_amount"`
	Currency             sql.NullString  `db:"currency"`
	PaymentMethod        sql.NullString  `db:"payment_method"`
	Status               string          `db:"status"`
	OcrRaw               sql.NullString  `db:"ocr_raw"`
	OcrJSON              sql.NullString  `db:"ocr_json"`
	TransactionID        sql.NullInt64   `db:"transaction_id"`
	MatchedTransactionID sql.NullInt64   `db:"matched_transaction_id"`
	MatchedReason        sql.NullString  `db:"matched_reason"`
	CreatedAt            string          `db:"created_at"`
}

func receiptToResponse(r receiptRow) models.ReceiptResponse {
	resp := models.ReceiptResponse{
		ID:        r.ID,
		ImagePath: r.ImagePath,
		ImageHash: r.ImageHash,
		Status:    r.Status,
		CreatedAt: r.CreatedAt,
		Items:     []models.ReceiptItemResponse{},
	}
	if r.MerchantName.Valid {
		resp.MerchantName = &r.MerchantName.String
	}
	if r.ReceiptDate.Valid {
		resp.ReceiptDate = &r.ReceiptDate.String
	}
	if r.ReceiptTime.Valid {
		resp.ReceiptTime = &r.ReceiptTime.String
	}
	if r.TotalAmount.Valid {
		resp.TotalAmount = &r.TotalAmount.Float64
	}
	if r.Currency.Valid {
		resp.Currency = &r.Currency.String
	}
	if r.PaymentMethod.Valid {
		resp.PaymentMethod = &r.PaymentMethod.String
	}
	if r.OcrRaw.Valid {
		resp.OcrRaw = &r.OcrRaw.String
	}
	if r.OcrJSON.Valid {
		resp.OcrJSON = &r.OcrJSON.String
	}
	if r.TransactionID.Valid {
		resp.TransactionID = &r.TransactionID.Int64
	}
	if r.MatchedTransactionID.Valid {
		resp.MatchedTransactionID = &r.MatchedTransactionID.Int64
	}
	if r.MatchedReason.Valid {
		resp.MatchedReason = &r.MatchedReason.String
	}
	return resp
}

type ReceiptHandler struct {
	db     *sqlx.DB
	cfg    *config.Config
	gemini *services.GeminiExtractService
}

func NewReceiptHandler(db *sqlx.DB, cfg *config.Config, gemini *services.GeminiExtractService) *ReceiptHandler {
	return &ReceiptHandler{db: db, cfg: cfg, gemini: gemini}
}

func (h *ReceiptHandler) Routes(r chi.Router) {
	r.Get("/api/receipts", h.List)
	r.Post("/api/receipts/upload", h.Upload)
	r.Post("/api/receipts/upload-batch", h.UploadBatch)
	r.Get("/api/receipts/by-transaction/{txnId}", h.GetByTransaction)
	r.Get("/api/receipts/{id}/image", h.GetImage)
	r.Get("/api/receipts/{id}", h.Get)
	r.Post("/api/receipts/{id}/confirm", h.Confirm)
	r.Patch("/api/receipts/items/{itemId}", h.UpdateItem)
}

func (h *ReceiptHandler) List(w http.ResponseWriter, r *http.Request) {
	var rows []receiptRow
	h.db.Select(&rows, `SELECT * FROM receipts ORDER BY created_at DESC`)

	matchIDs := []int{}
	for _, rr := range rows {
		if rr.MatchedTransactionID.Valid {
			matchIDs = append(matchIDs, int(rr.MatchedTransactionID.Int64))
		}
	}

	type matchedTxn struct {
		ID             int     `db:"id"`
		RawDate        string  `db:"raw_date"`
		Amount         float64 `db:"amount"`
		RawDescription string  `db:"raw_description"`
	}
	matchMap := map[int]matchedTxn{}
	if len(matchIDs) > 0 {
		query, args, _ := sqlx.In(`SELECT id, raw_date, amount, raw_description FROM transactions WHERE id IN (?)`, matchIDs)
		query = h.db.Rebind(query)
		var matched []matchedTxn
		h.db.Select(&matched, query, args...)
		for _, m := range matched {
			matchMap[m.ID] = m
		}
	}

	// Batch-load receipt items to avoid N+1
	receiptIDs := make([]int, len(rows))
	for i, rr := range rows {
		receiptIDs[i] = rr.ID
	}

	type batchItemRow struct {
		ReceiptID int             `db:"receipt_id"`
		ID        int             `db:"id"`
		Name      string          `db:"name"`
		Quantity  sql.NullFloat64 `db:"quantity"`
		UnitPrice sql.NullFloat64 `db:"unit_price"`
		LineTotal sql.NullFloat64 `db:"line_total"`
	}
	itemMap := map[int][]models.ReceiptItemResponse{}
	if len(receiptIDs) > 0 {
		inQuery, inArgs, _ := sqlx.In(`SELECT receipt_id, id, name, quantity, unit_price, line_total FROM receipt_items WHERE receipt_id IN (?)`, receiptIDs)
		inQuery = h.db.Rebind(inQuery)
		var allItems []batchItemRow
		if err := h.db.Select(&allItems, inQuery, inArgs...); err == nil {
			for _, item := range allItems {
				ri := models.ReceiptItemResponse{ID: item.ID, Name: item.Name}
				if item.Quantity.Valid {
					ri.Quantity = &item.Quantity.Float64
				}
				if item.UnitPrice.Valid {
					ri.UnitPrice = &item.UnitPrice.Float64
				}
				if item.LineTotal.Valid {
					ri.LineTotal = &item.LineTotal.Float64
				}
				itemMap[item.ReceiptID] = append(itemMap[item.ReceiptID], ri)
			}
		}
	}

	resp := make([]models.ReceiptResponse, 0, len(rows))
	for _, rr := range rows {
		receipt := receiptToResponse(rr)
		if rr.MatchedTransactionID.Valid {
			if mt, ok := matchMap[int(rr.MatchedTransactionID.Int64)]; ok {
				receipt.MatchedTransactionDate = &mt.RawDate
				receipt.MatchedTransactionAmount = &mt.Amount
				receipt.MatchedTransactionDescription = &mt.RawDescription
			}
		}
		if items, ok := itemMap[rr.ID]; ok {
			receipt.Items = items
		}
		resp = append(resp, receipt)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *ReceiptHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	receipt, err := h.getReceipt(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Receipt not found")
		return
	}
	writeJSON(w, http.StatusOK, receipt)
}

func (h *ReceiptHandler) GetByTransaction(w http.ResponseWriter, r *http.Request) {
	txnID, err := strconv.Atoi(chi.URLParam(r, "txnId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid transaction id")
		return
	}
	var receiptID int
	err = h.db.Get(&receiptID, `SELECT id FROM receipts WHERE transaction_id = ? LIMIT 1`, txnID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Receipt not found")
		return
	}
	receipt, err := h.getReceipt(receiptID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Receipt not found")
		return
	}
	writeJSON(w, http.StatusOK, receipt)
}

func (h *ReceiptHandler) GetImage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var imagePath string
	err = h.db.Get(&imagePath, `SELECT image_path FROM receipts WHERE id = ?`, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Receipt not found")
		return
	}
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "Image file not found on disk")
		return
	}
	ext := strings.ToLower(filepath.Ext(imagePath))
	mediaTypes := map[string]string{".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png", ".webp": "image/webp"}
	mt := mediaTypes[ext]
	if mt == "" {
		mt = "application/octet-stream"
	}
	w.Header().Set("Content-Type", mt)
	http.ServeFile(w, r, imagePath)
}

func (h *ReceiptHandler) Upload(w http.ResponseWriter, r *http.Request) {
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
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "image/jpeg"
	}
	imageHash := services.ComputeImageHash(content)

	var existID int
	err = h.db.Get(&existID, `SELECT id FROM receipts WHERE image_hash = ?`, imageHash)
	if err == nil {
		writeJSON(w, http.StatusOK, models.ReceiptUploadResult{
			Success: false, ReceiptID: &existID, Message: "This receipt was already uploaded", Errors: []string{},
		})
		return
	}

	safeName := filepath.Base(header.Filename)
	imagePath := filepath.Join(h.cfg.ReceiptsDir, imageHash+"_"+safeName)
	os.WriteFile(imagePath, content, 0o600)

	res, err := h.db.Exec(
		`INSERT INTO receipts (image_path, image_hash, status) VALUES (?, ?, 'pending')`, imagePath, imageHash)
	if err != nil {
		writeJSON(w, http.StatusOK, models.ReceiptUploadResult{Success: false, Message: "Failed to create receipt record", Errors: []string{}})
		return
	}
	receiptID := int(must64(res.LastInsertId()))
	svc := services.NewReceiptService(h.db, h.gemini)
	writeJSON(w, http.StatusOK, svc.ParseReceipt(receiptID, content, mimeType))
}

func (h *ReceiptHandler) UploadBatch(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	files := r.MultipartForm.File["files"]
	results := make([]models.ReceiptUploadResult, 0, len(files))
	for _, fh := range files {
		file, err := fh.Open()
		if err != nil {
			results = append(results, models.ReceiptUploadResult{Success: false, Message: "Failed to open file", Errors: []string{}})
			continue
		}
		content, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			results = append(results, models.ReceiptUploadResult{Success: false, Message: "Failed to read file", Errors: []string{}})
			continue
		}
		mimeType := fh.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = "image/jpeg"
		}
		imageHash := services.ComputeImageHash(content)
		var existID int
		err = h.db.Get(&existID, `SELECT id FROM receipts WHERE image_hash = ?`, imageHash)
		if err == nil {
			results = append(results, models.ReceiptUploadResult{Success: false, ReceiptID: &existID, Message: "This receipt was already uploaded", Errors: []string{}})
			continue
		}
		safeName := filepath.Base(fh.Filename)
		imagePath := filepath.Join(h.cfg.ReceiptsDir, imageHash+"_"+safeName)
		os.WriteFile(imagePath, content, 0o600)
		res, err := h.db.Exec(`INSERT INTO receipts (image_path, image_hash, status) VALUES (?, ?, 'pending')`, imagePath, imageHash)
		if err != nil {
			results = append(results, models.ReceiptUploadResult{Success: false, Message: "Failed to create receipt record", Errors: []string{}})
			continue
		}
		receiptID := int(must64(res.LastInsertId()))
		svc := services.NewReceiptService(h.db, h.gemini)
		results = append(results, svc.ParseReceipt(receiptID, content, mimeType))
	}
	writeJSON(w, http.StatusOK, results)
}

func (h *ReceiptHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var count int
	h.db.Get(&count, `SELECT COUNT(*) FROM receipts WHERE id = ?`, id)
	if count == 0 {
		writeError(w, http.StatusNotFound, "Receipt not found")
		return
	}
	var req models.ReceiptConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	svc := services.NewReceiptService(h.db, h.gemini)
	txnID, err := svc.ConfirmReceipt(id, req)
	if err != nil {
		log.Printf("receipt confirm: %v", err)
		writeError(w, http.StatusBadRequest, "receipt confirmation failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"transaction_id": txnID, "receipt_id": id})
}

func (h *ReceiptHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	itemID, err := strconv.Atoi(chi.URLParam(r, "itemId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item id")
		return
	}
	var count int
	h.db.Get(&count, `SELECT COUNT(*) FROM receipt_items WHERE id = ?`, itemID)
	if count == 0 {
		writeError(w, http.StatusNotFound, "Receipt item not found")
		return
	}
	var req models.ReceiptItemUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	var sets []string
	var args []interface{}
	if req.Name != nil {
		sets = append(sets, "name = ?")
		args = append(args, *req.Name)
	}
	if req.Quantity != nil {
		sets = append(sets, "quantity = ?")
		args = append(args, *req.Quantity)
	}
	if req.UnitPrice != nil {
		sets = append(sets, "unit_price = ?")
		args = append(args, *req.UnitPrice)
	}
	if req.LineTotal != nil {
		sets = append(sets, "line_total = ?")
		args = append(args, *req.LineTotal)
	}
	if len(sets) > 0 {
		args = append(args, itemID)
		h.db.Exec("UPDATE receipt_items SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"updated": true})
}

func (h *ReceiptHandler) loadReceiptItems(receiptID int) []models.ReceiptItemResponse {
	type itemRow struct {
		ID        int             `db:"id"`
		Name      string          `db:"name"`
		Quantity  sql.NullFloat64 `db:"quantity"`
		UnitPrice sql.NullFloat64 `db:"unit_price"`
		LineTotal sql.NullFloat64 `db:"line_total"`
	}
	var rows []itemRow
	h.db.Select(&rows, `SELECT id, name, quantity, unit_price, line_total FROM receipt_items WHERE receipt_id = ?`, receiptID)
	items := make([]models.ReceiptItemResponse, 0, len(rows))
	for _, row := range rows {
		item := models.ReceiptItemResponse{ID: row.ID, Name: row.Name}
		if row.Quantity.Valid {
			item.Quantity = &row.Quantity.Float64
		}
		if row.UnitPrice.Valid {
			item.UnitPrice = &row.UnitPrice.Float64
		}
		if row.LineTotal.Valid {
			item.LineTotal = &row.LineTotal.Float64
		}
		items = append(items, item)
	}
	return items
}

func (h *ReceiptHandler) getReceipt(id int) (models.ReceiptResponse, error) {
	var row receiptRow
	err := h.db.Get(&row, `SELECT * FROM receipts WHERE id = ?`, id)
	if err != nil {
		return models.ReceiptResponse{}, err
	}
	resp := receiptToResponse(row)
	resp.Items = h.loadReceiptItems(id)
	if row.MatchedTransactionID.Valid {
		type mt struct {
			RawDate        string  `db:"raw_date"`
			Amount         float64 `db:"amount"`
			RawDescription string  `db:"raw_description"`
		}
		var matched mt
		if err := h.db.Get(&matched, `SELECT raw_date, amount, raw_description FROM transactions WHERE id = ?`, row.MatchedTransactionID.Int64); err == nil {
			resp.MatchedTransactionDate = &matched.RawDate
			resp.MatchedTransactionAmount = &matched.Amount
			resp.MatchedTransactionDescription = &matched.RawDescription
		}
	}
	return resp, nil
}

func must64(v int64, _ error) int64 { return v }
