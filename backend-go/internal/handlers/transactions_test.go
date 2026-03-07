package handlers

import (
	"net/http"
	"testing"

	_ "modernc.org/sqlite"
)

func TestListTransactions_Empty(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupRouter(t, db)

	rr := doRequest(t, router, http.MethodGet, "/api/transactions", nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	parseJSON(t, rr, &resp)

	items, ok := resp["items"].([]interface{})
	if !ok {
		t.Fatal("expected 'items' array in response")
	}
	if len(items) != 0 {
		t.Errorf("expected empty items, got %d", len(items))
	}
	if resp["total"].(float64) != 0 {
		t.Errorf("expected total=0, got %v", resp["total"])
	}
}

func TestListTransactions_WithData(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupRouter(t, db)

	// Insert test data
	db.MustExec(`INSERT INTO transactions (source_hash, raw_date, raw_description, raw_amount, amount, category_source) VALUES
		('h1', '2025-01-01', 'TESCO', -25.00, -25.00, 'unclassified'),
		('h2', '2025-01-02', 'UBER', -15.00, -15.00, 'unclassified'),
		('h3', '2025-01-03', 'AMAZON', -9.99, -9.99, 'unclassified')`)

	rr := doRequest(t, router, http.MethodGet, "/api/transactions", nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	parseJSON(t, rr, &resp)

	total := int(resp["total"].(float64))
	if total != 3 {
		t.Errorf("expected total=3, got %d", total)
	}

	items := resp["items"].([]interface{})
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}

	// Check pagination
	if resp["page"].(float64) != 1 {
		t.Errorf("expected page=1, got %v", resp["page"])
	}
}

func TestListTransactions_Pagination(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupRouter(t, db)

	db.MustExec(`INSERT INTO transactions (source_hash, raw_date, raw_description, raw_amount, amount, category_source) VALUES
		('h1', '2025-01-01', 'TXN 1', -10.00, -10.00, 'unclassified'),
		('h2', '2025-01-02', 'TXN 2', -20.00, -20.00, 'unclassified'),
		('h3', '2025-01-03', 'TXN 3', -30.00, -30.00, 'unclassified')`)

	rr := doRequest(t, router, http.MethodGet, "/api/transactions?page=1&page_size=2", nil)

	var resp map[string]interface{}
	parseJSON(t, rr, &resp)

	items := resp["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("expected 2 items with page_size=2, got %d", len(items))
	}
	if resp["total_pages"].(float64) != 2 {
		t.Errorf("expected total_pages=2, got %v", resp["total_pages"])
	}
}

func TestGetTransaction(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupRouter(t, db)

	db.MustExec(`INSERT INTO transactions (id, source_hash, raw_date, raw_description, raw_amount, amount, category_source)
		VALUES (1, 'h1', '2025-01-01', 'TESCO STORES', -25.00, -25.00, 'unclassified')`)

	rr := doRequest(t, router, http.MethodGet, "/api/transactions/1", nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	parseJSON(t, rr, &resp)

	if resp["raw_description"].(string) != "TESCO STORES" {
		t.Errorf("expected raw_description='TESCO STORES', got %v", resp["raw_description"])
	}
}

func TestGetTransaction_NotFound(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupRouter(t, db)

	rr := doRequest(t, router, http.MethodGet, "/api/transactions/999", nil)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestUpdateTransaction(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupRouter(t, db)

	db.MustExec(`INSERT INTO categories (id, name) VALUES (1, 'Groceries')`)
	db.MustExec(`INSERT INTO transactions (id, source_hash, raw_date, raw_description, raw_amount, amount, category_source)
		VALUES (1, 'h1', '2025-01-01', 'TESCO STORES', -25.00, -25.00, 'unclassified')`)

	body := `{"category_id": 1}`
	rr := doRequest(t, router, http.MethodPatch, "/api/transactions/1", &body)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	parseJSON(t, rr, &resp)

	if resp["category_source"].(string) != "manual" {
		t.Errorf("expected category_source='manual' after update, got %v", resp["category_source"])
	}

	catID := resp["category_id"].(float64)
	if int(catID) != 1 {
		t.Errorf("expected category_id=1, got %v", catID)
	}
}

func TestUpdateTransaction_NotFound(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupRouter(t, db)

	body := `{"notes": "test"}`
	rr := doRequest(t, router, http.MethodPatch, "/api/transactions/999", &body)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestBulkClassify(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupRouter(t, db)

	db.MustExec(`INSERT INTO categories (id, name) VALUES (1, 'Groceries')`)
	db.MustExec(`INSERT INTO transactions (source_hash, raw_date, raw_description, raw_amount, amount, category_source) VALUES
		('h1', '2025-01-01', 'TXN 1', -10.00, -10.00, 'unclassified'),
		('h2', '2025-01-02', 'TXN 2', -20.00, -20.00, 'unclassified')`)

	body := `{"transaction_ids": [1, 2], "category_id": 1}`
	rr := doRequest(t, router, http.MethodPost, "/api/transactions/bulk-classify", &body)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	parseJSON(t, rr, &resp)

	updated := int(resp["updated"].(float64))
	if updated != 2 {
		t.Errorf("expected 2 updated, got %d", updated)
	}

	// Verify in DB
	var catSource string
	db.Get(&catSource, `SELECT category_source FROM transactions WHERE id = 1`)
	if catSource != "manual" {
		t.Errorf("expected category_source='manual', got %q", catSource)
	}
}

func TestBulkClassify_InvalidCategory(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupRouter(t, db)

	body := `{"transaction_ids": [1], "category_id": 999}`
	rr := doRequest(t, router, http.MethodPost, "/api/transactions/bulk-classify", &body)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestDeleteTransaction(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupRouter(t, db)

	db.MustExec(`INSERT INTO transactions (id, source_hash, raw_date, raw_description, raw_amount, amount, category_source)
		VALUES (1, 'h1', '2025-01-01', 'TXN 1', -10.00, -10.00, 'unclassified')`)

	rr := doRequest(t, router, http.MethodDelete, "/api/transactions/1", nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	// Verify deleted
	var count int
	db.Get(&count, `SELECT COUNT(*) FROM transactions WHERE id = 1`)
	if count != 0 {
		t.Error("expected transaction to be deleted")
	}
}

