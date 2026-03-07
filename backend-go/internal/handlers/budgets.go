package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/anthropics/accounting-tool/backend-go/internal/models"
)

type BudgetHandler struct {
	db *sqlx.DB
}

func NewBudgetHandler(db *sqlx.DB) *BudgetHandler {
	return &BudgetHandler{db: db}
}

func (h *BudgetHandler) Routes(r chi.Router) {
	r.Get("/api/budgets", h.List)
	r.Post("/api/budgets", h.Create)
	r.Patch("/api/budgets/{id}", h.Update)
	r.Delete("/api/budgets/{id}", h.Delete)
	r.Get("/api/budgets/status", h.Status)
}

func (h *BudgetHandler) List(w http.ResponseWriter, r *http.Request) {
	var rows []models.BudgetRow
	err := h.db.Select(&rows,
		`SELECT b.*, c.name AS category_name
		 FROM budgets b LEFT JOIN categories c ON b.category_id = c.id`)
	if err != nil {
		log.Printf("budgets list: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := make([]models.BudgetResponse, 0, len(rows))
	for _, row := range rows {
		resp = append(resp, budgetRowToResponse(row))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *BudgetHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.BudgetCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.MonthlyLimit <= 0 {
		writeError(w, http.StatusBadRequest, "monthly_limit must be positive")
		return
	}

	// Check category
	var catCount int
	h.db.Get(&catCount, `SELECT COUNT(*) FROM categories WHERE id = ?`, req.CategoryID)
	if catCount == 0 {
		writeError(w, http.StatusBadRequest, "Category not found")
		return
	}

	// Check existing
	var existCount int
	h.db.Get(&existCount, `SELECT COUNT(*) FROM budgets WHERE category_id = ?`, req.CategoryID)
	if existCount > 0 {
		writeError(w, http.StatusBadRequest, "Budget already exists for this category")
		return
	}

	res, err := h.db.Exec(
		`INSERT INTO budgets (category_id, monthly_limit) VALUES (?, ?)`,
		req.CategoryID, req.MonthlyLimit)
	if err != nil {
		log.Printf("budgets create: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	id, _ := res.LastInsertId()
	var row models.BudgetRow
	h.db.Get(&row,
		`SELECT b.*, c.name AS category_name
		 FROM budgets b LEFT JOIN categories c ON b.category_id = c.id
		 WHERE b.id = ?`, id)
	writeJSON(w, http.StatusOK, budgetRowToResponse(row))
}

func (h *BudgetHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req models.BudgetUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.MonthlyLimit <= 0 {
		writeError(w, http.StatusBadRequest, "monthly_limit must be positive")
		return
	}

	var count int
	h.db.Get(&count, `SELECT COUNT(*) FROM budgets WHERE id = ?`, id)
	if count == 0 {
		writeError(w, http.StatusNotFound, "Budget not found")
		return
	}

	if _, err := h.db.Exec(`UPDATE budgets SET monthly_limit = ? WHERE id = ?`, req.MonthlyLimit, id); err != nil {
		log.Printf("budgets update: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	var row models.BudgetRow
	h.db.Get(&row,
		`SELECT b.*, c.name AS category_name
		 FROM budgets b LEFT JOIN categories c ON b.category_id = c.id
		 WHERE b.id = ?`, id)
	writeJSON(w, http.StatusOK, budgetRowToResponse(row))
}

func (h *BudgetHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var count int
	h.db.Get(&count, `SELECT COUNT(*) FROM budgets WHERE id = ?`, id)
	if count == 0 {
		writeError(w, http.StatusNotFound, "Budget not found")
		return
	}

	if _, err := h.db.Exec(`DELETE FROM budgets WHERE id = ?`, id); err != nil {
		log.Printf("budgets delete: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

func (h *BudgetHandler) Status(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	monthLabel := monthStart.Format("2006-01")

	type budgetCat struct {
		ID           int     `db:"id"`
		CategoryID   int     `db:"category_id"`
		MonthlyLimit float64 `db:"monthly_limit"`
		CategoryName string  `db:"category_name"`
	}

	var budgets []budgetCat
	err := h.db.Select(&budgets,
		`SELECT b.id, b.category_id, b.monthly_limit, COALESCE(c.name, 'Unknown') AS category_name
		 FROM budgets b LEFT JOIN categories c ON b.category_id = c.id`)
	if err != nil {
		log.Printf("budgets status: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if len(budgets) == 0 {
		writeJSON(w, http.StatusOK, models.BudgetStatusResponse{
			Month: monthLabel,
			Items: []models.BudgetStatusItem{},
		})
		return
	}

	// Collect category IDs
	catIDs := make([]int, 0, len(budgets))
	for _, b := range budgets {
		catIDs = append(catIDs, b.CategoryID)
	}

	// Get spending per category this month
	query, args, _ := sqlx.In(
		`SELECT category_id, SUM(-amount) AS spent
		 FROM transactions
		 WHERE amount < 0 AND raw_date >= ? AND raw_date <= ? AND category_id IN (?)
		 GROUP BY category_id`,
		monthStart.Format("2006-01-02"), now.Format("2006-01-02"), catIDs)
	query = h.db.Rebind(query)

	type spendRow struct {
		CategoryID int     `db:"category_id"`
		Spent      float64 `db:"spent"`
	}
	var spendRows []spendRow
	h.db.Select(&spendRows, query, args...)

	spendMap := map[int]float64{}
	for _, s := range spendRows {
		spendMap[s.CategoryID] = s.Spent
	}

	items := make([]models.BudgetStatusItem, 0, len(budgets))
	for _, b := range budgets {
		spent := math.Round(spendMap[b.CategoryID]*100) / 100
		remaining := math.Round((b.MonthlyLimit-spent)*100) / 100
		pct := 0.0
		if b.MonthlyLimit > 0 {
			pct = math.Round((spent/b.MonthlyLimit)*1000) / 10
		}
		items = append(items, models.BudgetStatusItem{
			CategoryID:   b.CategoryID,
			CategoryName: b.CategoryName,
			MonthlyLimit: b.MonthlyLimit,
			Spent:        spent,
			Remaining:    remaining,
			Percent:      pct,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Percent > items[j].Percent
	})

	writeJSON(w, http.StatusOK, models.BudgetStatusResponse{
		Month: monthLabel,
		Items: items,
	})
}

func budgetRowToResponse(row models.BudgetRow) models.BudgetResponse {
	resp := models.BudgetResponse{
		ID:           row.ID,
		CategoryID:   row.CategoryID,
		MonthlyLimit: row.MonthlyLimit,
		CreatedAt:    row.CreatedAt.Format("2006-01-02T15:04:05"),
	}
	if row.CategoryName.Valid {
		resp.CategoryName = &row.CategoryName.String
	}
	return resp
}

func formatMoney(v float64) string {
	return fmt.Sprintf("%.2f", v)
}
