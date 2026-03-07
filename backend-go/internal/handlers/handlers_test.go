package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func setupHandlerTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dir := t.TempDir()
	dsn := filepath.Join(dir, "test.db") + "?_foreign_keys=on&_busy_timeout=5000&_journal_mode=WAL"
	db, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
		os.Remove(filepath.Join(dir, "test.db"))
	})

	db.MustExec(`
		CREATE TABLE accounts (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			bank TEXT NOT NULL DEFAULT '',
			account_type TEXT NOT NULL DEFAULT 'current',
			currency TEXT NOT NULL DEFAULT 'GBP',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE statements (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			account_id TEXT NOT NULL REFERENCES accounts(id),
			filename TEXT NOT NULL,
			file_hash TEXT UNIQUE NOT NULL,
			period_start DATE NOT NULL,
			period_end DATE NOT NULL,
			opening_balance REAL,
			closing_balance REAL,
			raw_text TEXT,
			imported_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE categories (
			id INTEGER PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			parent_id INTEGER,
			icon TEXT,
			color TEXT,
			is_expense BOOLEAN DEFAULT 1
		);
		CREATE TABLE transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			statement_id INTEGER,
			source_hash TEXT UNIQUE NOT NULL,
			raw_date DATE NOT NULL,
			raw_description TEXT NOT NULL,
			raw_amount REAL NOT NULL,
			raw_balance REAL,
			effective_date DATE,
			description TEXT,
			amount REAL NOT NULL,
			category_id INTEGER,
			category_source TEXT DEFAULT 'unclassified',
			is_excluded BOOLEAN DEFAULT 0,
			is_reconciled BOOLEAN DEFAULT 0,
			reconciled_at DATETIME,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE rules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pattern TEXT NOT NULL,
			pattern_type TEXT DEFAULT 'contains',
			category_id INTEGER NOT NULL,
			priority INTEGER DEFAULT 0,
			is_active BOOLEAN DEFAULT 1,
			created_from_txn_id INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE merchants (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			patterns TEXT NOT NULL DEFAULT '[]',
			category_id INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE budgets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			category_id INTEGER UNIQUE NOT NULL REFERENCES categories(id),
			monthly_limit REAL NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE receipts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			image_path TEXT NOT NULL,
			image_hash TEXT UNIQUE NOT NULL,
			merchant_name TEXT,
			receipt_date DATE,
			receipt_time TEXT,
			total_amount REAL,
			currency TEXT DEFAULT 'GBP',
			payment_method TEXT,
			status TEXT DEFAULT 'pending',
			ocr_raw TEXT,
			ocr_json TEXT,
			transaction_id INTEGER,
			matched_transaction_id INTEGER,
			matched_reason TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE receipt_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			receipt_id INTEGER NOT NULL REFERENCES receipts(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			quantity REAL,
			unit_price REAL,
			line_total REAL
		);
	`)
	return db
}

func setupRouter(t *testing.T, db *sqlx.DB) chi.Router {
	t.Helper()
	r := chi.NewRouter()
	NewCategoryHandler(db).Routes(r)
	NewTransactionHandler(db).Routes(r)
	NewRuleHandler(db).Routes(r)
	NewBudgetHandler(db).Routes(r)
	return r
}

func doRequest(t *testing.T, router chi.Router, method, path string, body *string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, strings.NewReader(*body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func parseJSON(t *testing.T, rr *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	if err := json.NewDecoder(rr.Body).Decode(v); err != nil {
		t.Fatalf("failed to parse JSON response: %v\nbody: %s", err, rr.Body.String())
	}
}

