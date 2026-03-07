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
)

type CategoryHandler struct {
	db *sqlx.DB
}

func NewCategoryHandler(db *sqlx.DB) *CategoryHandler {
	return &CategoryHandler{db: db}
}

func (h *CategoryHandler) Routes(r chi.Router) {
	r.Get("/api/categories", h.List)
	r.Get("/api/categories/tree", h.Tree)
	r.Post("/api/categories", h.Create)
	r.Delete("/api/categories/{id}", h.Delete)
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	var cats []models.Category
	if err := h.db.Select(&cats, `SELECT * FROM categories ORDER BY name`); err != nil {
		log.Printf("categories list: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := make([]models.CategoryResponse, 0, len(cats))
	for _, c := range cats {
		resp = append(resp, catToResponse(c))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *CategoryHandler) Tree(w http.ResponseWriter, r *http.Request) {
	var cats []models.Category
	if err := h.db.Select(&cats, `SELECT * FROM categories`); err != nil {
		log.Printf("categories tree: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Build lookup of children
	children := map[int64][]models.Category{}
	var roots []models.Category
	for _, c := range cats {
		if c.ParentID.Valid {
			children[c.ParentID.Int64] = append(children[c.ParentID.Int64], c)
		} else {
			roots = append(roots, c)
		}
	}

	var buildNode func(c models.Category) models.CategoryResponse
	buildNode = func(c models.Category) models.CategoryResponse {
		resp := catToResponse(c)
		for _, child := range children[int64(c.ID)] {
			resp.Children = append(resp.Children, buildNode(child))
		}
		if resp.Children == nil {
			resp.Children = []models.CategoryResponse{}
		}
		return resp
	}

	result := make([]models.CategoryResponse, 0, len(roots))
	for _, root := range roots {
		result = append(result, buildNode(root))
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CategoryCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Check duplicate name
	var count int
	h.db.Get(&count, `SELECT COUNT(*) FROM categories WHERE name = ?`, req.Name)
	if count > 0 {
		writeError(w, http.StatusBadRequest, "Category name already exists")
		return
	}

	// Verify parent
	if req.ParentID != nil {
		var parentCount int
		h.db.Get(&parentCount, `SELECT COUNT(*) FROM categories WHERE id = ?`, *req.ParentID)
		if parentCount == 0 {
			writeError(w, http.StatusBadRequest, "Parent category not found")
			return
		}
	}

	isExpense := true
	if req.IsExpense != nil {
		isExpense = *req.IsExpense
	}

	res, err := h.db.Exec(
		`INSERT INTO categories (name, parent_id, icon, color, is_expense) VALUES (?, ?, ?, ?, ?)`,
		req.Name, req.ParentID, req.Icon, req.Color, isExpense)
	if err != nil {
		log.Printf("categories create: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	id, _ := res.LastInsertId()
	var cat models.Category
	h.db.Get(&cat, `SELECT * FROM categories WHERE id = ?`, id)

	writeJSON(w, http.StatusOK, catToResponse(cat))
}

func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var cat models.Category
	if err := h.db.Get(&cat, `SELECT * FROM categories WHERE id = ?`, id); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Category not found")
			return
		}
		log.Printf("categories delete get: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Check transactions
	var txnCount int
	h.db.Get(&txnCount, `SELECT COUNT(*) FROM transactions WHERE category_id = ?`, id)
	if txnCount > 0 {
		writeError(w, http.StatusBadRequest, strconv.Itoa(txnCount)+" transactions use this category")
		return
	}

	// Check children
	var childCount int
	h.db.Get(&childCount, `SELECT COUNT(*) FROM categories WHERE parent_id = ?`, id)
	if childCount > 0 {
		writeError(w, http.StatusBadRequest, "Cannot delete: category has child categories")
		return
	}

	if _, err := h.db.Exec(`DELETE FROM categories WHERE id = ?`, id); err != nil {
		log.Printf("categories delete: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

func catToResponse(c models.Category) models.CategoryResponse {
	resp := models.CategoryResponse{
		ID:        c.ID,
		Name:      c.Name,
		IsExpense: c.IsExpense,
		Children:  []models.CategoryResponse{},
	}
	if c.ParentID.Valid {
		v := c.ParentID.Int64
		resp.ParentID = &v
	}
	if c.Icon.Valid {
		resp.Icon = &c.Icon.String
	}
	if c.Color.Valid {
		resp.Color = &c.Color.String
	}
	return resp
}
