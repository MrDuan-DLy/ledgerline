package services

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	// Use a temp file DB to avoid in-memory shared-cache locking issues
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

func TestMatches(t *testing.T) {
	tests := []struct {
		name     string
		rule     cachedRule
		desc     string
		expected bool
	}{
		{
			name:     "exact match case insensitive",
			rule:     cachedRule{Pattern: "tesco", PatternType: "exact"},
			desc:     "TESCO",
			expected: true,
		},
		{
			name:     "exact match no match",
			rule:     cachedRule{Pattern: "tesco", PatternType: "exact"},
			desc:     "TESCO EXTRA",
			expected: false,
		},
		{
			name:     "contains match",
			rule:     cachedRule{Pattern: "uber", PatternType: "contains"},
			desc:     "UBER EATS LONDON",
			expected: true,
		},
		{
			name:     "contains no match",
			rule:     cachedRule{Pattern: "uber", PatternType: "contains"},
			desc:     "AMAZON PRIME",
			expected: false,
		},
		{
			name: "regex match",
			rule: func() cachedRule {
				r := cachedRule{Pattern: `^AMAZON.*`, PatternType: "regex"}
				// Compile like the real code does
				re, _ := compileRegex(r.Pattern)
				r.compiledRE = re
				return r
			}(),
			desc:     "AMAZON PRIME MEMBERSHIP",
			expected: true,
		},
		{
			name: "regex no match",
			rule: func() cachedRule {
				r := cachedRule{Pattern: `^AMAZON.*`, PatternType: "regex"}
				re, _ := compileRegex(r.Pattern)
				r.compiledRE = re
				return r
			}(),
			desc:     "BUY FROM AMAZON",
			expected: false,
		},
		{
			name:     "regex with nil compiled pattern returns false",
			rule:     cachedRule{Pattern: `[invalid`, PatternType: "regex", compiledRE: nil},
			desc:     "ANYTHING",
			expected: false,
		},
		{
			name:     "unknown pattern type returns false",
			rule:     cachedRule{Pattern: "test", PatternType: "unknown"},
			desc:     "TEST",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := matches(tc.rule, tc.desc)
			if got != tc.expected {
				t.Errorf("matches() = %v, want %v", got, tc.expected)
			}
		})
	}
}

// compileRegex is a test helper to mirror the service's compile behavior.
func compileRegex(pattern string) (*regexp.Regexp, error) {
	return regexp.Compile("(?i)" + pattern)
}

func TestClassifyService_Classify(t *testing.T) {
	db := setupTestDB(t)

	// Insert categories
	db.MustExec(`INSERT INTO categories (id, name) VALUES (1, 'Groceries'), (2, 'Transport'), (3, 'Entertainment')`)

	// Insert rules with different priorities
	db.MustExec(`INSERT INTO rules (pattern, pattern_type, category_id, priority, is_active) VALUES
		('TESCO', 'contains', 1, 10, 1),
		('UBER', 'contains', 2, 5, 1),
		('UBER EATS', 'contains', 3, 20, 1)`)

	svc := NewClassifyService(db)

	t.Run("contains match returns correct category", func(t *testing.T) {
		catID, source := svc.Classify("TESCO STORES 1234")
		if catID == nil {
			t.Fatal("expected category match, got nil")
		}
		if *catID != 1 {
			t.Errorf("expected category 1 (Groceries), got %d", *catID)
		}
		if source != "rule" {
			t.Errorf("expected source 'rule', got %q", source)
		}
	})

	t.Run("higher priority rule wins", func(t *testing.T) {
		// "UBER EATS" matches both "UBER" (priority 5) and "UBER EATS" (priority 20)
		catID, source := svc.Classify("UBER EATS LONDON")
		if catID == nil {
			t.Fatal("expected category match, got nil")
		}
		if *catID != 3 {
			t.Errorf("expected category 3 (Entertainment), got %d — higher priority rule should win", *catID)
		}
		if source != "rule" {
			t.Errorf("expected source 'rule', got %q", source)
		}
	})

	t.Run("unclassified when no rules match", func(t *testing.T) {
		catID, source := svc.Classify("RANDOM PAYMENT XYZ")
		if catID != nil {
			t.Errorf("expected nil category for unmatched description, got %d", *catID)
		}
		if source != "unclassified" {
			t.Errorf("expected source 'unclassified', got %q", source)
		}
	})
}

func TestClassifyService_ReclassifyAll(t *testing.T) {
	db := setupTestDB(t)

	db.MustExec(`INSERT INTO categories (id, name) VALUES (1, 'Groceries'), (2, 'Transport')`)
	db.MustExec(`INSERT INTO rules (pattern, pattern_type, category_id, priority, is_active) VALUES
		('TESCO', 'contains', 1, 10, 1),
		('UBER', 'contains', 2, 5, 1)`)

	// Insert transactions: some rule-classified, some manual, some unclassified
	db.MustExec(`INSERT INTO transactions (source_hash, raw_date, raw_description, raw_amount, amount, category_id, category_source) VALUES
		('hash1', '2025-01-01', 'TESCO STORES', -25.00, -25.00, NULL, 'unclassified'),
		('hash2', '2025-01-02', 'UBER TRIP', -15.00, -15.00, NULL, 'unclassified'),
		('hash3', '2025-01-03', 'MANUAL ENTRY', -10.00, -10.00, 1, 'manual'),
		('hash4', '2025-01-04', 'RANDOM SHOP', -5.00, -5.00, NULL, 'unclassified')`)

	svc := NewClassifyService(db)
	updated, err := svc.ReclassifyAll()
	if err != nil {
		t.Fatalf("ReclassifyAll() error: %v", err)
	}

	// hash1 → Groceries (1), hash2 → Transport (2), hash4 → no change (still unclassified)
	if updated != 2 {
		t.Errorf("expected 2 updated, got %d", updated)
	}

	// Verify hash1 was classified
	var catID1 *int
	db.Get(&catID1, `SELECT category_id FROM transactions WHERE source_hash = 'hash1'`)
	if catID1 == nil || *catID1 != 1 {
		t.Errorf("hash1: expected category_id=1, got %v", catID1)
	}

	// Verify hash2 was classified
	var catID2 *int
	db.Get(&catID2, `SELECT category_id FROM transactions WHERE source_hash = 'hash2'`)
	if catID2 == nil || *catID2 != 2 {
		t.Errorf("hash2: expected category_id=2, got %v", catID2)
	}

	// Verify manual transaction was NOT touched
	var catSource string
	db.Get(&catSource, `SELECT category_source FROM transactions WHERE source_hash = 'hash3'`)
	if catSource != "manual" {
		t.Errorf("hash3: expected category_source='manual', got %q", catSource)
	}
	var manualCatID int
	db.Get(&manualCatID, `SELECT category_id FROM transactions WHERE source_hash = 'hash3'`)
	if manualCatID != 1 {
		t.Errorf("hash3: expected category_id=1 (unchanged), got %d", manualCatID)
	}
}

func TestClassifyService_InvalidateCache(t *testing.T) {
	db := setupTestDB(t)

	db.MustExec(`INSERT INTO categories (id, name) VALUES (1, 'Groceries')`)
	db.MustExec(`INSERT INTO rules (pattern, pattern_type, category_id, priority, is_active) VALUES
		('TESCO', 'contains', 1, 10, 1)`)

	svc := NewClassifyService(db)

	// First classify loads rules
	catID, _ := svc.Classify("TESCO")
	if catID == nil || *catID != 1 {
		t.Fatal("expected initial classify to work")
	}

	// Add a new rule directly to DB
	db.MustExec(`INSERT INTO categories (id, name) VALUES (2, 'Transport')`)
	db.MustExec(`INSERT INTO rules (pattern, pattern_type, category_id, priority, is_active) VALUES
		('UBER', 'contains', 2, 5, 1)`)

	// Without invalidation, new rule shouldn't be picked up (rules are cached)
	catID2, _ := svc.Classify("UBER TRIP")
	if catID2 != nil {
		// The cache still has old rules, so UBER won't match
		t.Logf("NOTE: cache was auto-reloaded, catID=%d", *catID2)
	}

	// Invalidate and re-classify
	svc.InvalidateCache()
	catID3, source2 := svc.Classify("UBER TRIP")
	if catID3 == nil {
		t.Fatal("expected UBER to match after cache invalidation")
	}
	if *catID3 != 2 {
		t.Errorf("expected category 2 (Transport), got %d", *catID3)
	}
	if source2 != "rule" {
		t.Errorf("expected source 'rule', got %q", source2)
	}
}
