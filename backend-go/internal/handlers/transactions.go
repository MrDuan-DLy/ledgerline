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
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/anthropics/accounting-tool/backend-go/internal/models"
)

type TransactionHandler struct {
	db *sqlx.DB
}

func NewTransactionHandler(db *sqlx.DB) *TransactionHandler {
	return &TransactionHandler{db: db}
}

func (h *TransactionHandler) Routes(r chi.Router) {
	r.Get("/api/transactions", h.List)
	r.Get("/api/transactions/stats/summary", h.Summary)
	r.Get("/api/transactions/stats/series", h.Series)
	r.Get("/api/transactions/stats/monthly", h.Monthly)
	r.Get("/api/transactions/stats/pace", h.Pace)
	r.Get("/api/transactions/{id}", h.Get)
	r.Patch("/api/transactions/{id}", h.Update)
	r.Delete("/api/transactions/{id}", h.Delete)
	r.Post("/api/transactions/bulk-delete", h.BulkDelete)
	r.Post("/api/transactions/bulk-exclude", h.BulkExclude)
	r.Post("/api/transactions/bulk-classify", h.BulkClassify)
}

func (h *TransactionHandler) getTransferCategoryIDs() []int {
	var ids []int
	h.db.Select(&ids, `SELECT id FROM categories WHERE name IN ('Transfer In', 'Transfer Out')`)
	return ids
}

func excludeTransferClause(transferIDs []int) (string, []interface{}) {
	clause := "t.is_excluded = 0"
	var args []interface{}
	if len(transferIDs) > 0 {
		placeholders := make([]string, len(transferIDs))
		for i, id := range transferIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		clause += fmt.Sprintf(" AND (t.category_id IS NULL OR t.category_id NOT IN (%s))",
			strings.Join(placeholders, ","))
	}
	return clause, args
}

// ---- List ----

func (h *TransactionHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	var whereClauses []string
	var args []interface{}

	if q.Get("excluded_only") == "true" {
		whereClauses = append(whereClauses, "t.is_excluded = 1")
	} else if q.Get("hide_excluded") == "true" {
		whereClauses = append(whereClauses, "t.is_excluded = 0")
	}
	if v := q.Get("start_date"); v != "" {
		whereClauses = append(whereClauses, "t.raw_date >= ?")
		args = append(args, v)
	}
	if v := q.Get("end_date"); v != "" {
		whereClauses = append(whereClauses, "t.raw_date <= ?")
		args = append(args, v)
	}
	if v := q.Get("category_id"); v != "" {
		whereClauses = append(whereClauses, "t.category_id = ?")
		args = append(args, v)
	}
	if q.Get("unclassified_only") == "true" {
		whereClauses = append(whereClauses, "t.category_id IS NULL")
	}
	if v := q.Get("search"); v != "" {
		whereClauses = append(whereClauses, "t.raw_description LIKE ?")
		args = append(args, "%"+v+"%")
	}
	if v := q.Get("statement_id"); v != "" {
		whereClauses = append(whereClauses, "t.statement_id = ?")
		args = append(args, v)
	}

	where := ""
	if len(whereClauses) > 0 {
		where = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Count total
	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM transactions t %s`, where)
	if err := h.db.Get(&total, countQuery, args...); err != nil {
		log.Printf("transaction list count: %v", err)
	}

	offset := (page - 1) * pageSize
	totalPages := (total + pageSize - 1) / pageSize

	query := fmt.Sprintf(
		`SELECT t.*, c.name AS category_name
		 FROM transactions t
		 LEFT JOIN categories c ON t.category_id = c.id
		 %s ORDER BY t.raw_date DESC LIMIT ? OFFSET ?`, where)
	queryArgs := append(args, pageSize, offset)

	var rows []models.TransactionRow
	h.db.Select(&rows, query, queryArgs...)

	items := make([]models.TransactionResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, txnRowToResponse(row))
	}

	writeJSON(w, http.StatusOK, models.TransactionListResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

// ---- Get ----

func (h *TransactionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var row models.TransactionRow
	err = h.db.Get(&row,
		`SELECT t.*, c.name AS category_name
		 FROM transactions t LEFT JOIN categories c ON t.category_id = c.id
		 WHERE t.id = ?`, id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Transaction not found")
			return
		}
		log.Printf("transaction get: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, txnRowToResponse(row))
}

// ---- Update ----

func (h *TransactionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var count int
	h.db.Get(&count, `SELECT COUNT(*) FROM transactions WHERE id = ?`, id)
	if count == 0 {
		writeError(w, http.StatusNotFound, "Transaction not found")
		return
	}

	var req models.TransactionUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	sets := []string{}
	var args []interface{}

	if req.EffectiveDate != nil {
		sets = append(sets, "effective_date = ?")
		args = append(args, *req.EffectiveDate)
	}
	if req.Description != nil {
		sets = append(sets, "description = ?")
		args = append(args, *req.Description)
	}
	if req.CategoryID != nil {
		var catCount int
		h.db.Get(&catCount, `SELECT COUNT(*) FROM categories WHERE id = ?`, *req.CategoryID)
		if catCount == 0 {
			writeError(w, http.StatusBadRequest, "Category not found")
			return
		}
		sets = append(sets, "category_id = ?", "category_source = 'manual'")
		args = append(args, *req.CategoryID)
	}
	if req.Notes != nil {
		sets = append(sets, "notes = ?")
		args = append(args, *req.Notes)
	}
	if req.IsExcluded != nil {
		sets = append(sets, "is_excluded = ?")
		args = append(args, *req.IsExcluded)
	}

	if len(sets) > 0 {
		sets = append(sets, "updated_at = ?")
		args = append(args, time.Now().Format("2006-01-02T15:04:05"))
		args = append(args, id)
		q := fmt.Sprintf(`UPDATE transactions SET %s WHERE id = ?`, strings.Join(sets, ", "))
		if _, err := h.db.Exec(q, args...); err != nil {
			log.Printf("transaction update: %v", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
	}

	var row models.TransactionRow
	h.db.Get(&row,
		`SELECT t.*, c.name AS category_name
		 FROM transactions t LEFT JOIN categories c ON t.category_id = c.id
		 WHERE t.id = ?`, id)
	writeJSON(w, http.StatusOK, txnRowToResponse(row))
}

// ---- Delete ----

func (h *TransactionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var count int
	h.db.Get(&count, `SELECT COUNT(*) FROM transactions WHERE id = ?`, id)
	if count == 0 {
		writeError(w, http.StatusNotFound, "Transaction not found")
		return
	}

	if _, err := h.db.Exec(`DELETE FROM transactions WHERE id = ?`, id); err != nil {
		log.Printf("transaction delete: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Transaction %d deleted.", id),
	})
}

// ---- Bulk Delete ----

func (h *TransactionHandler) BulkDelete(w http.ResponseWriter, r *http.Request) {
	var req models.BulkDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if len(req.TransactionIDs) == 0 {
		writeJSON(w, http.StatusOK, map[string]int{"deleted": 0})
		return
	}

	query, args, err := sqlx.In(`DELETE FROM transactions WHERE id IN (?)`, req.TransactionIDs)
	if err != nil {
		log.Printf("transaction bulk delete: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	query = h.db.Rebind(query)
	res, err := h.db.Exec(query, args...)
	if err != nil {
		log.Printf("transaction bulk delete exec: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	deleted, _ := res.RowsAffected()
	writeJSON(w, http.StatusOK, map[string]int64{"deleted": deleted})
}

// ---- Bulk Exclude ----

func (h *TransactionHandler) BulkExclude(w http.ResponseWriter, r *http.Request) {
	var req models.BulkExcludeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if len(req.TransactionIDs) == 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{"updated": 0, "is_excluded": req.IsExcluded})
		return
	}

	query, args, err := sqlx.In(`UPDATE transactions SET is_excluded = ? WHERE id IN (?)`,
		req.IsExcluded, req.TransactionIDs)
	if err != nil {
		log.Printf("transaction bulk exclude: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	query = h.db.Rebind(query)
	res, err := h.db.Exec(query, args...)
	if err != nil {
		log.Printf("transaction bulk exclude exec: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	updated, _ := res.RowsAffected()
	writeJSON(w, http.StatusOK, map[string]interface{}{"updated": updated, "is_excluded": req.IsExcluded})
}

// ---- Bulk Classify ----

func (h *TransactionHandler) BulkClassify(w http.ResponseWriter, r *http.Request) {
	var req models.BulkClassifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	var catCount int
	h.db.Get(&catCount, `SELECT COUNT(*) FROM categories WHERE id = ?`, req.CategoryID)
	if catCount == 0 {
		writeError(w, http.StatusBadRequest, "Category not found")
		return
	}

	if len(req.TransactionIDs) == 0 {
		writeJSON(w, http.StatusOK, map[string]int{"updated": 0})
		return
	}

	query, args, err := sqlx.In(
		`UPDATE transactions SET category_id = ?, category_source = 'manual' WHERE id IN (?)`,
		req.CategoryID, req.TransactionIDs)
	if err != nil {
		log.Printf("transaction bulk classify: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	query = h.db.Rebind(query)
	res, err := h.db.Exec(query, args...)
	if err != nil {
		log.Printf("transaction bulk classify exec: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	updated, _ := res.RowsAffected()
	writeJSON(w, http.StatusOK, map[string]int64{"updated": updated})
}

// ---- Stats Summary ----

func (h *TransactionHandler) Summary(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var whereParts []string
	var args []interface{}

	if v := q.Get("start_date"); v != "" {
		whereParts = append(whereParts, "t.raw_date >= ?")
		args = append(args, v)
	}
	if v := q.Get("end_date"); v != "" {
		whereParts = append(whereParts, "t.raw_date <= ?")
		args = append(args, v)
	}

	transferIDs := h.getTransferCategoryIDs()
	excl, exclArgs := excludeTransferClause(transferIDs)
	whereParts = append(whereParts, excl)
	args = append(args, exclArgs...)

	where := "WHERE " + strings.Join(whereParts, " AND ")

	type summaryRow struct {
		Count        int             `db:"cnt"`
		Income       sql.NullFloat64 `db:"income"`
		Expenses     sql.NullFloat64 `db:"expenses"`
		Unclassified int             `db:"unclassified"`
	}

	var row summaryRow
	query := fmt.Sprintf(
		`SELECT
			COUNT(*) AS cnt,
			SUM(CASE WHEN t.amount > 0 THEN t.amount ELSE 0 END) AS income,
			SUM(CASE WHEN t.amount < 0 THEN t.amount ELSE 0 END) AS expenses,
			SUM(CASE WHEN t.category_id IS NULL THEN 1 ELSE 0 END) AS unclassified
		 FROM transactions t %s`, where)
	h.db.Get(&row, query, args...)

	income := 0.0
	if row.Income.Valid {
		income = row.Income.Float64
	}
	expenses := 0.0
	if row.Expenses.Valid {
		expenses = row.Expenses.Float64
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total_transactions": row.Count,
		"total_income":       math.Round(income*100) / 100,
		"total_expenses":     math.Round(math.Abs(expenses)*100) / 100,
		"net":                math.Round((income+expenses)*100) / 100,
		"unclassified_count": row.Unclassified,
	})
}

// ---- Stats Series ----

func (h *TransactionHandler) Series(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	startDate := q.Get("start_date")
	endDate := q.Get("end_date")

	var whereParts []string
	var args []interface{}

	if startDate != "" {
		whereParts = append(whereParts, "t.raw_date >= ?")
		args = append(args, startDate)
	}
	if endDate != "" {
		whereParts = append(whereParts, "t.raw_date <= ?")
		args = append(args, endDate)
	}

	transferIDs := h.getTransferCategoryIDs()
	excl, exclArgs := excludeTransferClause(transferIDs)
	whereParts = append(whereParts, excl)
	args = append(args, exclArgs...)

	where := "WHERE " + strings.Join(whereParts, " AND ")

	// Daily series
	dailyQuery := fmt.Sprintf(
		`SELECT
			t.raw_date AS date,
			SUM(t.amount) AS net,
			SUM(CASE WHEN t.amount > 0 THEN t.amount ELSE 0 END) AS income,
			SUM(CASE WHEN t.amount < 0 THEN -t.amount ELSE 0 END) AS expenses,
			COUNT(*) AS count
		 FROM transactions t %s
		 GROUP BY t.raw_date ORDER BY t.raw_date`, where)

	type dailyRow struct {
		Date     string  `db:"date"`
		Net      float64 `db:"net"`
		Income   float64 `db:"income"`
		Expenses float64 `db:"expenses"`
		Count    int     `db:"count"`
	}

	var dailyRows []dailyRow
	h.db.Select(&dailyRows, dailyQuery, args...)

	daily := make([]models.DailySeriesPoint, 0, len(dailyRows))
	cumulative := 0.0
	for _, dr := range dailyRows {
		cumulative += dr.Net
		daily = append(daily, models.DailySeriesPoint{
			Date:       dr.Date,
			Net:        math.Round(dr.Net*100) / 100,
			Income:     math.Round(dr.Income*100) / 100,
			Expenses:   math.Round(dr.Expenses*100) / 100,
			Count:      dr.Count,
			Cumulative: math.Round(cumulative*100) / 100,
		})
	}

	// Category breakdown
	catQuery := fmt.Sprintf(
		`SELECT
			c.id AS category_id,
			COALESCE(c.name, 'Unclassified') AS category_name,
			SUM(CASE WHEN t.amount < 0 THEN -t.amount ELSE 0 END) AS expenses,
			SUM(CASE WHEN t.amount > 0 THEN t.amount ELSE 0 END) AS income,
			SUM(t.amount) AS net,
			COUNT(t.id) AS count
		 FROM transactions t LEFT JOIN categories c ON t.category_id = c.id
		 %s GROUP BY c.id, c.name ORDER BY SUM(t.amount) ASC`, where)

	type catRow struct {
		CategoryID   sql.NullInt64 `db:"category_id"`
		CategoryName string        `db:"category_name"`
		Expenses     float64       `db:"expenses"`
		Income       float64       `db:"income"`
		Net          float64       `db:"net"`
		Count        int           `db:"count"`
	}

	var catRows []catRow
	h.db.Select(&catRows, catQuery, args...)

	categories := make([]models.CategoryTotal, 0, len(catRows))
	for _, cr := range catRows {
		var catID *int64
		if cr.CategoryID.Valid {
			catID = &cr.CategoryID.Int64
		}
		categories = append(categories, models.CategoryTotal{
			CategoryID:   catID,
			CategoryName: cr.CategoryName,
			Expenses:     math.Round(cr.Expenses*100) / 100,
			Income:       math.Round(cr.Income*100) / 100,
			Net:          math.Round(cr.Net*100) / 100,
			Count:        cr.Count,
		})
	}

	resp := models.StatsSeriesResponse{
		Daily:      daily,
		Categories: categories,
	}
	if startDate != "" {
		resp.StartDate = &startDate
	}
	if endDate != "" {
		resp.EndDate = &endDate
	}

	writeJSON(w, http.StatusOK, resp)
}

// ---- Stats Monthly ----

func (h *TransactionHandler) Monthly(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	startDate := q.Get("start_date")
	endDate := q.Get("end_date")
	categoryIDStr := q.Get("category_id")

	var whereParts []string
	var args []interface{}

	if startDate != "" {
		whereParts = append(whereParts, "t.raw_date >= ?")
		args = append(args, startDate)
	}
	if endDate != "" {
		whereParts = append(whereParts, "t.raw_date <= ?")
		args = append(args, endDate)
	}
	if categoryIDStr != "" {
		whereParts = append(whereParts, "t.category_id = ?")
		args = append(args, categoryIDStr)
	}

	transferIDs := h.getTransferCategoryIDs()
	excl, exclArgs := excludeTransferClause(transferIDs)
	whereParts = append(whereParts, excl)
	args = append(args, exclArgs...)
	whereParts = append(whereParts, "t.amount < 0")

	where := "WHERE " + strings.Join(whereParts, " AND ")

	query := fmt.Sprintf(
		`SELECT
			strftime('%%Y-%%m', t.raw_date) AS month,
			SUM(-t.amount) AS total_expenses
		 FROM transactions t %s
		 GROUP BY strftime('%%Y-%%m', t.raw_date)
		 ORDER BY strftime('%%Y-%%m', t.raw_date)`, where)

	type monthRow struct {
		Month         string  `db:"month"`
		TotalExpenses float64 `db:"total_expenses"`
	}

	var rows []monthRow
	h.db.Select(&rows, query, args...)

	series := make([]models.MonthlySpendPoint, 0, len(rows))
	for _, row := range rows {
		series = append(series, models.MonthlySpendPoint{
			Month:         row.Month,
			TotalExpenses: math.Round(row.TotalExpenses*100) / 100,
		})
	}

	resp := models.MonthlySpendResponse{
		Series: series,
	}
	if startDate != "" {
		resp.StartDate = &startDate
	}
	if endDate != "" {
		resp.EndDate = &endDate
	}
	if categoryIDStr != "" {
		v, _ := strconv.ParseInt(categoryIDStr, 10, 64)
		resp.CategoryID = &v
	}

	writeJSON(w, http.StatusOK, resp)
}

// ---- Stats Pace ----

func (h *TransactionHandler) Pace(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	curStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	prevEnd := curStart.AddDate(0, 0, -1)
	prevStart := time.Date(prevEnd.Year(), prevEnd.Month(), 1, 0, 0, 0, 0, time.Local)

	transferIDs := h.getTransferCategoryIDs()

	dailyCum := func(mStart, mEnd time.Time) []models.PaceDayPoint {
		var whereParts []string
		var args []interface{}
		whereParts = append(whereParts, "t.raw_date >= ?", "t.raw_date <= ?", "t.amount < 0")
		args = append(args, mStart.Format("2006-01-02"), mEnd.Format("2006-01-02"))
		excl, exclArgs := excludeTransferClause(transferIDs)
		whereParts = append(whereParts, excl)
		args = append(args, exclArgs...)

		where := "WHERE " + strings.Join(whereParts, " AND ")
		query := fmt.Sprintf(
			`SELECT t.raw_date AS dt, SUM(-t.amount) AS spend
			 FROM transactions t %s
			 GROUP BY t.raw_date ORDER BY t.raw_date`, where)

		type row struct {
			Dt    string  `db:"dt"`
			Spend float64 `db:"spend"`
		}
		var rows []row
		h.db.Select(&rows, query, args...)

		dayMap := map[string]float64{}
		for _, rr := range rows {
			dayMap[rr.Dt] = rr.Spend
		}

		numDays := int(mEnd.Sub(mStart).Hours()/24) + 1
		points := make([]models.PaceDayPoint, 0, numDays)
		cum := 0.0
		for i := 0; i < numDays; i++ {
			d := mStart.AddDate(0, 0, i)
			ds := d.Format("2006-01-02")
			cum += dayMap[ds]
			points = append(points, models.PaceDayPoint{
				Day:        d.Day(),
				Cumulative: math.Round(cum*100) / 100,
			})
		}
		return points
	}

	// Current month up to today
	curEnd := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	writeJSON(w, http.StatusOK, models.SpendingPaceResponse{
		CurrentMonth:  curStart.Format("2006-01"),
		CurrentSeries: dailyCum(curStart, curEnd),
		PrevMonth:     prevStart.Format("2006-01"),
		PrevSeries:    dailyCum(prevStart, prevEnd),
	})
}

// ---- Helpers ----

func txnRowToResponse(row models.TransactionRow) models.TransactionResponse {
	resp := models.TransactionResponse{
		ID:             row.ID,
		SourceHash:     row.SourceHash,
		RawDate:        row.RawDate,
		RawDescription: row.RawDescription,
		RawAmount:      row.RawAmount,
		Amount:         row.Amount,
		CategorySource: row.CategorySource,
		IsExcluded:     row.IsExcluded,
		IsReconciled:   row.IsReconciled,
		CreatedAt:      row.CreatedAt.Format("2006-01-02T15:04:05"),
		UpdatedAt:      row.UpdatedAt.Format("2006-01-02T15:04:05"),
	}
	if row.StatementID.Valid {
		resp.StatementID = &row.StatementID.Int64
	}
	if row.RawBalance.Valid {
		resp.RawBalance = &row.RawBalance.Float64
	}
	if row.EffectiveDate.Valid {
		resp.EffectiveDate = &row.EffectiveDate.String
	}
	if row.Description.Valid {
		resp.Description = &row.Description.String
	}
	if row.CategoryID.Valid {
		resp.CategoryID = &row.CategoryID.Int64
	}
	if row.CategoryName.Valid {
		resp.CategoryName = &row.CategoryName.String
	}
	if row.ReconciledAt.Valid {
		s := row.ReconciledAt.Time.Format("2006-01-02T15:04:05")
		resp.ReconciledAt = &s
	}
	if row.Notes.Valid {
		resp.Notes = &row.Notes.String
	}
	return resp
}
