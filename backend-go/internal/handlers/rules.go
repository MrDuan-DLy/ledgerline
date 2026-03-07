package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/anthropics/accounting-tool/backend-go/internal/models"
	"github.com/anthropics/accounting-tool/backend-go/internal/services"
)

type RuleHandler struct {
	db *sqlx.DB
}

func NewRuleHandler(db *sqlx.DB) *RuleHandler {
	return &RuleHandler{db: db}
}

func (h *RuleHandler) Routes(r chi.Router) {
	r.Get("/api/rules", h.List)
	r.Post("/api/rules", h.Create)
	r.Delete("/api/rules/{id}", h.Delete)
	r.Patch("/api/rules/{id}/toggle", h.Toggle)
	r.Post("/api/rules/reclassify", h.Reclassify)
}

func (h *RuleHandler) List(w http.ResponseWriter, r *http.Request) {
	var rows []models.RuleRow
	err := h.db.Select(&rows,
		`SELECT r.*, c.name AS category_name
		 FROM rules r LEFT JOIN categories c ON r.category_id = c.id
		 ORDER BY r.priority DESC, r.pattern`)
	if err != nil {
		log.Printf("rules list: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := make([]models.RuleResponse, 0, len(rows))
	for _, row := range rows {
		resp = append(resp, ruleRowToResponse(row))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *RuleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.RuleCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.PatternType == "" {
		req.PatternType = "contains"
	}

	// Verify category
	var catCount int
	h.db.Get(&catCount, `SELECT COUNT(*) FROM categories WHERE id = ?`, req.CategoryID)
	if catCount == 0 {
		writeError(w, http.StatusBadRequest, "Category not found")
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	res, err := h.db.Exec(
		`INSERT INTO rules (pattern, pattern_type, category_id, priority, is_active, created_from_txn_id)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		req.Pattern, req.PatternType, req.CategoryID, req.Priority, isActive, req.CreatedFromTxnID)
	if err != nil {
		log.Printf("rules create: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	id, _ := res.LastInsertId()
	var row models.RuleRow
	h.db.Get(&row,
		`SELECT r.*, c.name AS category_name
		 FROM rules r LEFT JOIN categories c ON r.category_id = c.id
		 WHERE r.id = ?`, id)

	writeJSON(w, http.StatusOK, ruleRowToResponse(row))
}

func (h *RuleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var count int
	h.db.Get(&count, `SELECT COUNT(*) FROM rules WHERE id = ?`, id)
	if count == 0 {
		writeError(w, http.StatusNotFound, "Rule not found")
		return
	}

	if _, err := h.db.Exec(`DELETE FROM rules WHERE id = ?`, id); err != nil {
		log.Printf("rules delete: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

func (h *RuleHandler) Toggle(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var rule models.Rule
	if err := h.db.Get(&rule, `SELECT * FROM rules WHERE id = ?`, id); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Rule not found")
			return
		}
		log.Printf("rules toggle: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	newActive := !rule.IsActive
	if _, err := h.db.Exec(`UPDATE rules SET is_active = ? WHERE id = ?`, newActive, id); err != nil {
		log.Printf("rules toggle update: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"is_active": newActive})
}

func (h *RuleHandler) Reclassify(w http.ResponseWriter, r *http.Request) {
	svc := services.NewClassifyService(h.db)
	updated, err := svc.ReclassifyAll()
	if err != nil {
		log.Printf("rules reclassify: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"updated": updated})
}

func ruleRowToResponse(row models.RuleRow) models.RuleResponse {
	resp := models.RuleResponse{
		ID:          row.ID,
		Pattern:     row.Pattern,
		PatternType: row.PatternType,
		CategoryID:  row.CategoryID,
		Priority:    row.Priority,
		IsActive:    row.IsActive,
		CreatedAt:   row.CreatedAt.Format("2006-01-02T15:04:05"),
	}
	if row.CategoryName.Valid {
		resp.CategoryName = &row.CategoryName.String
	}
	if row.CreatedFromTxnID.Valid {
		v := int(row.CreatedFromTxnID.Int64)
		resp.CreatedFromTxnID = &v
	}
	return resp
}
