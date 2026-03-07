package handlers

import (
	"net/http"
	"testing"

	_ "modernc.org/sqlite"
)

func TestCategoryList_Empty(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupRouter(t, db)

	rr := doRequest(t, router, http.MethodGet, "/api/categories", nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp []interface{}
	parseJSON(t, rr, &resp)

	if len(resp) != 0 {
		t.Errorf("expected empty list, got %d items", len(resp))
	}
}

func TestCategoryCreate(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupRouter(t, db)

	body := `{"name": "Groceries"}`
	rr := doRequest(t, router, http.MethodPost, "/api/categories", &body)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	parseJSON(t, rr, &resp)

	if resp["name"].(string) != "Groceries" {
		t.Errorf("expected name='Groceries', got %v", resp["name"])
	}
	if resp["is_expense"].(bool) != true {
		t.Error("expected is_expense=true by default")
	}
}

func TestCategoryCreateDuplicate(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupRouter(t, db)

	body := `{"name": "Groceries"}`
	doRequest(t, router, http.MethodPost, "/api/categories", &body)

	// Try creating again
	rr := doRequest(t, router, http.MethodPost, "/api/categories", &body)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for duplicate, got %d", rr.Code)
	}
}

func TestCategoryCreate_WithOptions(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupRouter(t, db)

	body := `{"name": "Income", "is_expense": false, "icon": "money", "color": "#00ff00"}`
	rr := doRequest(t, router, http.MethodPost, "/api/categories", &body)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	parseJSON(t, rr, &resp)

	if resp["is_expense"].(bool) != false {
		t.Error("expected is_expense=false")
	}
	if resp["icon"].(string) != "money" {
		t.Errorf("expected icon='money', got %v", resp["icon"])
	}
}

func TestCategoryDelete(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupRouter(t, db)

	db.MustExec(`INSERT INTO categories (id, name) VALUES (1, 'Groceries')`)

	rr := doRequest(t, router, http.MethodDelete, "/api/categories/1", nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	// Verify deleted
	var count int
	db.Get(&count, `SELECT COUNT(*) FROM categories WHERE id = 1`)
	if count != 0 {
		t.Error("expected category to be deleted")
	}
}

func TestCategoryDelete_NotFound(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupRouter(t, db)

	rr := doRequest(t, router, http.MethodDelete, "/api/categories/999", nil)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestCategoryDelete_WithTransactions(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupRouter(t, db)

	db.MustExec(`INSERT INTO categories (id, name) VALUES (1, 'Groceries')`)
	db.MustExec(`INSERT INTO transactions (source_hash, raw_date, raw_description, raw_amount, amount, category_id, category_source)
		VALUES ('h1', '2025-01-01', 'TESCO', -25.00, -25.00, 1, 'rule')`)

	rr := doRequest(t, router, http.MethodDelete, "/api/categories/1", nil)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when transactions exist, got %d", rr.Code)
	}
}

func TestRuleList_Empty(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupRouter(t, db)

	rr := doRequest(t, router, http.MethodGet, "/api/rules", nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp []interface{}
	parseJSON(t, rr, &resp)

	if len(resp) != 0 {
		t.Errorf("expected empty list, got %d items", len(resp))
	}
}

func TestRuleCreate(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupRouter(t, db)

	db.MustExec(`INSERT INTO categories (id, name) VALUES (1, 'Groceries')`)

	body := `{"pattern": "TESCO", "pattern_type": "contains", "category_id": 1, "priority": 10}`
	rr := doRequest(t, router, http.MethodPost, "/api/rules", &body)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	parseJSON(t, rr, &resp)

	if resp["pattern"].(string) != "TESCO" {
		t.Errorf("expected pattern='TESCO', got %v", resp["pattern"])
	}
	if int(resp["priority"].(float64)) != 10 {
		t.Errorf("expected priority=10, got %v", resp["priority"])
	}
}

func TestRuleCreate_InvalidCategory(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupRouter(t, db)

	body := `{"pattern": "TEST", "category_id": 999}`
	rr := doRequest(t, router, http.MethodPost, "/api/rules", &body)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}
