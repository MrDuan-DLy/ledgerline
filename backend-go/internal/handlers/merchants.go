package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/anthropics/accounting-tool/backend-go/internal/models"
	"github.com/anthropics/accounting-tool/backend-go/internal/services"
)

type MerchantHandler struct {
	db *sqlx.DB
}

func NewMerchantHandler(db *sqlx.DB) *MerchantHandler {
	return &MerchantHandler{db: db}
}

func (h *MerchantHandler) Routes(r chi.Router) {
	r.Get("/api/merchants", h.List)
	r.Get("/api/merchants/match", h.Match)
	r.Post("/api/merchants", h.Create)
	r.Post("/api/merchants/backfill", h.Backfill)
	r.Patch("/api/merchants/{id}", h.Update)
	r.Delete("/api/merchants/{id}", h.Delete)
	r.Post("/api/merchants/{id}/merge", h.Merge)
	r.Get("/api/merchants/{id}/transactions", h.Transactions)
}

type merchantDBRow struct {
	ID         int           `db:"id"`
	Name       string        `db:"name"`
	Patterns   string        `db:"patterns"`
	CategoryID sql.NullInt64 `db:"category_id"`
	CreatedAt  string        `db:"created_at"`
}

func merchantRowToResponse(r merchantDBRow) models.MerchantResponse {
	resp := models.MerchantResponse{
		ID:        r.ID,
		Name:      r.Name,
		CreatedAt: r.CreatedAt,
		Patterns:  []string{},
	}
	json.Unmarshal([]byte(r.Patterns), &resp.Patterns)
	if resp.Patterns == nil {
		resp.Patterns = []string{}
	}
	if r.CategoryID.Valid {
		resp.CategoryID = &r.CategoryID.Int64
	}
	return resp
}

func (h *MerchantHandler) List(w http.ResponseWriter, r *http.Request) {
	withCounts := r.URL.Query().Get("with_counts") == "true"

	var rows []merchantDBRow
	h.db.Select(&rows, `SELECT id, name, patterns, category_id, created_at FROM merchants ORDER BY name`)

	resp := make([]models.MerchantResponse, 0, len(rows))
	for _, row := range rows {
		mr := merchantRowToResponse(row)
		if withCounts {
			searchTerms := []string{services.Normalize(row.Name)}
			var patterns []string
			json.Unmarshal([]byte(row.Patterns), &patterns)
			seen := map[string]bool{searchTerms[0]: true}
			for _, p := range patterns {
				if p != "" && !seen[p] {
					seen[p] = true
					searchTerms = append(searchTerms, p)
				}
			}

			count := 0
			if len(searchTerms) > 0 {
				var conditions []string
				var args []interface{}
				for _, term := range searchTerms {
					if term != "" {
						conditions = append(conditions, "raw_description LIKE ?")
						args = append(args, "%"+term+"%")
					}
				}
				if len(conditions) > 0 {
					q := "SELECT COUNT(*) FROM transactions WHERE " + strings.Join(conditions, " OR ")
					h.db.Get(&count, q, args...)
				}
			}
			mr.TransactionCount = &count
		}
		resp = append(resp, mr)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *MerchantHandler) Match(w http.ResponseWriter, r *http.Request) {
	rawName := r.URL.Query().Get("raw_name")
	svc := services.NewMerchantService(h.db)
	merchantID, merchantName, score, matchType := svc.Match(rawName)

	resp := models.MerchantMatchResponse{
		Score:     score,
		MatchType: matchType,
	}
	if merchantName != nil {
		resp.CanonicalName = merchantName
	}
	if merchantID != nil {
		resp.MerchantID = merchantID
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *MerchantHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.MerchantCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	var count int
	h.db.Get(&count, `SELECT COUNT(*) FROM merchants WHERE name = ?`, req.Name)
	if count > 0 {
		writeError(w, http.StatusConflict, "Merchant with this name already exists")
		return
	}

	var normalizedPatterns []string
	for _, p := range req.Patterns {
		norm := services.Normalize(p)
		if norm != "" {
			normalizedPatterns = append(normalizedPatterns, norm)
		}
	}
	if normalizedPatterns == nil {
		normalizedPatterns = []string{}
	}
	patternsJSON, _ := json.Marshal(normalizedPatterns)

	res, err := h.db.Exec(
		`INSERT INTO merchants (name, patterns, category_id) VALUES (?, ?, ?)`,
		req.Name, string(patternsJSON), req.CategoryID)
	if err != nil {
		log.Printf("merchant create: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	id, _ := res.LastInsertId()
	var row merchantDBRow
	h.db.Get(&row, `SELECT id, name, patterns, category_id, created_at FROM merchants WHERE id = ?`, id)
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, http.StatusCreated, merchantRowToResponse(row))
}

func (h *MerchantHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var count int
	h.db.Get(&count, `SELECT COUNT(*) FROM merchants WHERE id = ?`, id)
	if count == 0 {
		writeError(w, http.StatusNotFound, "Merchant not found")
		return
	}

	var req models.MerchantUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Name != nil {
		var conflictCount int
		h.db.Get(&conflictCount, `SELECT COUNT(*) FROM merchants WHERE name = ? AND id != ?`, *req.Name, id)
		if conflictCount > 0 {
			writeError(w, http.StatusConflict, "Merchant with this name already exists")
			return
		}
		if _, err := h.db.Exec(`UPDATE merchants SET name = ? WHERE id = ?`, *req.Name, id); err != nil {
			log.Printf("merchant update name: %v", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
	}

	if req.Patterns != nil {
		var normalizedPatterns []string
		for _, p := range req.Patterns {
			norm := services.Normalize(p)
			if norm != "" {
				normalizedPatterns = append(normalizedPatterns, norm)
			}
		}
		if normalizedPatterns == nil {
			normalizedPatterns = []string{}
		}
		patternsJSON, _ := json.Marshal(normalizedPatterns)
		if _, err := h.db.Exec(`UPDATE merchants SET patterns = ? WHERE id = ?`, string(patternsJSON), id); err != nil {
			log.Printf("merchant update patterns: %v", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
	}

	if req.CategoryID != nil {
		if _, err := h.db.Exec(`UPDATE merchants SET category_id = ? WHERE id = ?`, *req.CategoryID, id); err != nil {
			log.Printf("merchant update category: %v", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
	}

	var row merchantDBRow
	h.db.Get(&row, `SELECT id, name, patterns, category_id, created_at FROM merchants WHERE id = ?`, id)
	writeJSON(w, http.StatusOK, merchantRowToResponse(row))
}

func (h *MerchantHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var name string
	err = h.db.Get(&name, `SELECT name FROM merchants WHERE id = ?`, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Merchant not found")
		return
	}

	if _, err := h.db.Exec(`DELETE FROM merchants WHERE id = ?`, id); err != nil {
		log.Printf("merchant delete: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Merchant '%s' deleted.", name),
	})
}

func (h *MerchantHandler) Merge(w http.ResponseWriter, r *http.Request) {
	targetID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req models.MerchantMergeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.SourceMerchantID == targetID {
		writeError(w, http.StatusBadRequest, "Cannot merge a merchant into itself")
		return
	}

	var targetRow merchantDBRow
	err = h.db.Get(&targetRow, `SELECT id, name, patterns, category_id, created_at FROM merchants WHERE id = ?`, targetID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Target merchant not found")
		return
	}

	var sourceRow merchantDBRow
	err = h.db.Get(&sourceRow, `SELECT id, name, patterns, category_id, created_at FROM merchants WHERE id = ?`, req.SourceMerchantID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Source merchant not found")
		return
	}

	// Absorb patterns
	var targetPatterns, sourcePatterns []string
	json.Unmarshal([]byte(targetRow.Patterns), &targetPatterns)
	json.Unmarshal([]byte(sourceRow.Patterns), &sourcePatterns)

	seen := map[string]bool{}
	for _, p := range targetPatterns {
		seen[p] = true
	}
	for _, p := range sourcePatterns {
		if !seen[p] {
			targetPatterns = append(targetPatterns, p)
			seen[p] = true
		}
	}
	sourceNorm := services.Normalize(sourceRow.Name)
	if !seen[sourceNorm] {
		targetPatterns = append(targetPatterns, sourceNorm)
	}

	patternsJSON, _ := json.Marshal(targetPatterns)
	if _, err := h.db.Exec(`UPDATE merchants SET patterns = ? WHERE id = ?`, string(patternsJSON), targetID); err != nil {
		log.Printf("merchant merge update: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if _, err := h.db.Exec(`DELETE FROM merchants WHERE id = ?`, req.SourceMerchantID); err != nil {
		log.Printf("merchant merge delete source: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	h.db.Get(&targetRow, `SELECT id, name, patterns, category_id, created_at FROM merchants WHERE id = ?`, targetID)
	writeJSON(w, http.StatusOK, merchantRowToResponse(targetRow))
}

func (h *MerchantHandler) Transactions(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var row merchantDBRow
	err = h.db.Get(&row, `SELECT id, name, patterns, category_id, created_at FROM merchants WHERE id = ?`, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Merchant not found")
		return
	}

	searchTerms := []string{services.Normalize(row.Name)}
	var patterns []string
	json.Unmarshal([]byte(row.Patterns), &patterns)
	seen := map[string]bool{searchTerms[0]: true}
	for _, p := range patterns {
		if p != "" && !seen[p] {
			seen[p] = true
			searchTerms = append(searchTerms, p)
		}
	}

	if len(searchTerms) == 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"merchant": merchantRowToResponse(row), "total_spend": 0, "count": 0, "transactions": []interface{}{},
		})
		return
	}

	var conditions []string
	var args []interface{}
	for _, term := range searchTerms {
		if term != "" {
			conditions = append(conditions, "t.raw_description LIKE ?")
			args = append(args, "%"+term+"%")
		}
	}

	q := r.URL.Query()
	dateFilter := ""
	if v := q.Get("start_date"); v != "" {
		dateFilter += " AND t.raw_date >= ?"
		args = append(args, v)
	}
	if v := q.Get("end_date"); v != "" {
		dateFilter += " AND t.raw_date <= ?"
		args = append(args, v)
	}

	query := fmt.Sprintf(
		`SELECT t.id, t.raw_date, t.raw_description, t.amount, COALESCE(c.name, '') AS category_name
		 FROM transactions t LEFT JOIN categories c ON t.category_id = c.id
		 WHERE (%s) %s ORDER BY t.raw_date DESC`,
		strings.Join(conditions, " OR "), dateFilter)

	type txnItem struct {
		ID             int     `db:"id" json:"id"`
		RawDate        string  `db:"raw_date" json:"date"`
		RawDescription string  `db:"raw_description" json:"description"`
		Amount         float64 `db:"amount" json:"amount"`
		CategoryName   string  `db:"category_name" json:"category_name"`
	}

	var txns []txnItem
	h.db.Select(&txns, query, args...)

	totalSpend := 0.0
	for _, t := range txns {
		if t.Amount < 0 {
			totalSpend += math.Abs(t.Amount)
		}
	}

	if txns == nil {
		txns = []txnItem{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"merchant":     merchantRowToResponse(row),
		"total_spend":  math.Round(totalSpend*100) / 100,
		"count":        len(txns),
		"transactions": txns,
	})
}

func (h *MerchantHandler) Backfill(w http.ResponseWriter, r *http.Request) {
	svc := services.NewMerchantService(h.db)

	type txnRow struct {
		ID             int            `db:"id"`
		RawDescription string         `db:"raw_description"`
		Description    sql.NullString `db:"description"`
	}

	var txns []txnRow
	h.db.Select(&txns, `SELECT id, raw_description, description FROM transactions`)

	updated := 0
	patternsLearned := 0

	for _, txn := range txns {
		if txn.Description.Valid && txn.Description.String != txn.RawDescription {
			continue
		}

		merchantID, merchantName, _, matchType := svc.Match(txn.RawDescription)
		if merchantID == nil || matchType == "none" {
			continue
		}

		if _, err := h.db.Exec(`UPDATE transactions SET description = ? WHERE id = ?`, *merchantName, txn.ID); err != nil {
			log.Printf("merchant backfill update txn: %v", err)
			continue
		}

		norm := services.Normalize(txn.RawDescription)
		if norm != "" {
			// Check if pattern already exists
			var existingPatterns string
			h.db.Get(&existingPatterns, `SELECT patterns FROM merchants WHERE id = ?`, *merchantID)
			var patterns []string
			json.Unmarshal([]byte(existingPatterns), &patterns)
			found := false
			for _, p := range patterns {
				if p == norm {
					found = true
					break
				}
			}
			if !found {
				patterns = append(patterns, norm)
				pj, _ := json.Marshal(patterns)
				if _, err := h.db.Exec(`UPDATE merchants SET patterns = ? WHERE id = ?`, string(pj), *merchantID); err != nil {
					log.Printf("merchant backfill update patterns: %v", err)
				} else {
					patternsLearned++
				}
			}
		}
		updated++
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":              true,
		"transactions_updated": updated,
		"patterns_learned":     patternsLearned,
		"total_scanned":        len(txns),
		"message":              fmt.Sprintf("Updated %d transactions, learned %d new patterns.", updated, patternsLearned),
	})
}
