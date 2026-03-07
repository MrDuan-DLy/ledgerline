package handlers

import (
	"database/sql"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/anthropics/accounting-tool/backend-go/internal/config"
	"github.com/anthropics/accounting-tool/backend-go/internal/models"
	"github.com/anthropics/accounting-tool/backend-go/internal/parsers"
	"github.com/anthropics/accounting-tool/backend-go/internal/services"
)

type StatementHandler struct {
	db     *sqlx.DB
	cfg    *config.Config
	gemini *services.GeminiExtractService
}

func NewStatementHandler(db *sqlx.DB, cfg *config.Config, gemini *services.GeminiExtractService) *StatementHandler {
	return &StatementHandler{db: db, cfg: cfg, gemini: gemini}
}

func (h *StatementHandler) Routes(r chi.Router) {
	r.Get("/api/statements", h.List)
	r.Post("/api/statements/upload", h.Upload)
	r.Get("/api/statements/{id}", h.Get)
}

func (h *StatementHandler) List(w http.ResponseWriter, r *http.Request) {
	type stmtRow struct {
		ID             int             `db:"id"`
		AccountID      string          `db:"account_id"`
		Filename       string          `db:"filename"`
		FileHash       string          `db:"file_hash"`
		PeriodStart    string          `db:"period_start"`
		PeriodEnd      string          `db:"period_end"`
		OpeningBalance sql.NullFloat64 `db:"opening_balance"`
		ClosingBalance sql.NullFloat64 `db:"closing_balance"`
		ImportedAt     string          `db:"imported_at"`
		TxnCount       int             `db:"txn_count"`
	}

	var rows []stmtRow
	h.db.Select(&rows,
		`SELECT s.id, s.account_id, s.filename, s.file_hash, s.period_start, s.period_end,
		        s.opening_balance, s.closing_balance, s.imported_at,
		        (SELECT COUNT(*) FROM transactions t WHERE t.statement_id = s.id) AS txn_count
		 FROM statements s ORDER BY s.imported_at DESC`)

	resp := make([]models.StatementResponse, 0, len(rows))
	for _, row := range rows {
		sr := models.StatementResponse{
			ID:          row.ID,
			AccountID:   row.AccountID,
			Filename:    row.Filename,
			FileHash:    row.FileHash,
			PeriodStart: row.PeriodStart,
			PeriodEnd:   row.PeriodEnd,
			ImportedAt:  row.ImportedAt,
			TxnCount:    row.TxnCount,
		}
		if row.OpeningBalance.Valid {
			sr.OpeningBalance = &row.OpeningBalance.Float64
		}
		if row.ClosingBalance.Valid {
			sr.ClosingBalance = &row.ClosingBalance.Float64
		}
		resp = append(resp, sr)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *StatementHandler) Upload(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusInternalServerError, "failed to read file")
		return
	}

	filename := header.Filename
	lower := strings.ToLower(filename)

	var parser parsers.Parser
	var accountID, accountName, bank string

	switch {
	case strings.HasSuffix(lower, ".pdf"):
		extractFn := func(data []byte) (bool, map[string]interface{}, string) {
			ok, d, errMsg, _ := h.gemini.ExtractFromPDF(data)
			return ok, d, errMsg
		}
		parser = parsers.NewGeminiPDFParser(extractFn)
		accountID = "pdf-main"
		accountName = "PDF Import Account"
		bank = "PDF"
	case strings.HasSuffix(lower, ".csv") && strings.Contains(lower, "starling"):
		parser = &parsers.StarlingCSVParser{}
		accountID = "csv-main"
		accountName = "CSV Import Account"
		bank = "CSV"
	case strings.HasSuffix(lower, ".csv"):
		writeJSON(w, http.StatusOK, models.ImportResult{
			Success: false,
			Message: "Unknown CSV format. Supported: Starling (filename must contain 'starling')",
		})
		return
	default:
		writeJSON(w, http.StatusOK, models.ImportResult{
			Success: false,
			Message: "Only PDF and CSV files are supported",
		})
		return
	}

	importSvc := services.NewImportService(h.db)
	fileHash := services.ComputeFileHash(content)

	if importSvc.CheckDuplicateFile(fileHash) {
		writeJSON(w, http.StatusOK, models.ImportResult{
			Success: false,
			Message: "This statement was already imported",
		})
		return
	}

	importSvc.EnsureAccount(accountID, accountName, bank, "current", h.cfg.DefaultCurrency)

	parsed, err := parser.Parse(content)
	if err != nil {
		writeJSON(w, http.StatusOK, models.ImportResult{
			Success: false,
			Message: "Failed to parse file: " + err.Error(),
		})
		return
	}

	// Save file
	safeName := filepath.Base(filename)
	uploadPath := filepath.Join(h.cfg.UploadsDir, fileHash+"_"+safeName)
	if err := writeFile(uploadPath, content); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save file")
		return
	}

	stmtID, err := importSvc.CreateStatement(
		accountID, filename, fileHash,
		parsed.PeriodStart, parsed.PeriodEnd,
		parsed.OpeningBalance, parsed.ClosingBalance,
		parsed.RawText)
	if err != nil {
		writeJSON(w, http.StatusOK, models.ImportResult{
			Success: false,
			Message: "Failed to create statement record",
		})
		return
	}

	var categoryMap map[string]int
	if parser != nil && strings.HasSuffix(lower, ".csv") {
		categoryMap = services.BuildCategoryMap(h.db)
	}

	result := importSvc.ImportTransactions(accountID, stmtID, parsed.Transactions, categoryMap)
	writeJSON(w, http.StatusOK, result)
}

func (h *StatementHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	type stmtRow struct {
		ID             int             `db:"id"`
		AccountID      string          `db:"account_id"`
		Filename       string          `db:"filename"`
		FileHash       string          `db:"file_hash"`
		PeriodStart    string          `db:"period_start"`
		PeriodEnd      string          `db:"period_end"`
		OpeningBalance sql.NullFloat64 `db:"opening_balance"`
		ClosingBalance sql.NullFloat64 `db:"closing_balance"`
		ImportedAt     string          `db:"imported_at"`
		TxnCount       int             `db:"txn_count"`
	}

	var row stmtRow
	err = h.db.Get(&row,
		`SELECT s.id, s.account_id, s.filename, s.file_hash, s.period_start, s.period_end,
		        s.opening_balance, s.closing_balance, s.imported_at,
		        (SELECT COUNT(*) FROM transactions t WHERE t.statement_id = s.id) AS txn_count
		 FROM statements s WHERE s.id = ?`, id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Statement not found")
			return
		}
		log.Printf("statement get: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	sr := models.StatementResponse{
		ID:          row.ID,
		AccountID:   row.AccountID,
		Filename:    row.Filename,
		FileHash:    row.FileHash,
		PeriodStart: row.PeriodStart,
		PeriodEnd:   row.PeriodEnd,
		ImportedAt:  row.ImportedAt,
		TxnCount:    row.TxnCount,
	}
	if row.OpeningBalance.Valid {
		sr.OpeningBalance = &row.OpeningBalance.Float64
	}
	if row.ClosingBalance.Valid {
		sr.ClosingBalance = &row.ClosingBalance.Float64
	}
	writeJSON(w, http.StatusOK, sr)
}

func writeFile(path string, content []byte) error {
	return writeFileBytes(path, content)
}
