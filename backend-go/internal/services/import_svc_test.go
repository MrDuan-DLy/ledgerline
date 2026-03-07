package services

import (
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/anthropics/accounting-tool/backend-go/internal/parsers"
)

func TestComputeTransactionHash(t *testing.T) {
	date := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	bal := 1000.50

	t.Run("same inputs produce same hash", func(t *testing.T) {
		h1 := ComputeTransactionHash(date, "TESCO", -25.00, &bal)
		h2 := ComputeTransactionHash(date, "TESCO", -25.00, &bal)
		if h1 != h2 {
			t.Errorf("same inputs produced different hashes: %q vs %q", h1, h2)
		}
	})

	t.Run("different descriptions produce different hashes", func(t *testing.T) {
		h1 := ComputeTransactionHash(date, "TESCO", -25.00, &bal)
		h2 := ComputeTransactionHash(date, "ASDA", -25.00, &bal)
		if h1 == h2 {
			t.Error("different descriptions should produce different hashes")
		}
	})

	t.Run("different amounts produce different hashes", func(t *testing.T) {
		h1 := ComputeTransactionHash(date, "TESCO", -25.00, &bal)
		h2 := ComputeTransactionHash(date, "TESCO", -30.00, &bal)
		if h1 == h2 {
			t.Error("different amounts should produce different hashes")
		}
	})

	t.Run("different dates produce different hashes", func(t *testing.T) {
		date2 := time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC)
		h1 := ComputeTransactionHash(date, "TESCO", -25.00, &bal)
		h2 := ComputeTransactionHash(date2, "TESCO", -25.00, &bal)
		if h1 == h2 {
			t.Error("different dates should produce different hashes")
		}
	})

	t.Run("nil balance vs non-nil balance produce different hashes", func(t *testing.T) {
		h1 := ComputeTransactionHash(date, "TESCO", -25.00, nil)
		h2 := ComputeTransactionHash(date, "TESCO", -25.00, &bal)
		if h1 == h2 {
			t.Error("nil balance vs non-nil balance should produce different hashes")
		}
	})

	t.Run("hash is valid SHA256 hex", func(t *testing.T) {
		h := ComputeTransactionHash(date, "TEST", -10.00, nil)
		if len(h) != 64 {
			t.Errorf("expected 64-char hex string, got %d chars: %q", len(h), h)
		}
	})
}

func TestComputeFileHash(t *testing.T) {
	content := []byte("hello world")
	expected := fmt.Sprintf("%x", sha256.Sum256(content))

	got := ComputeFileHash(content)
	if got != expected {
		t.Errorf("ComputeFileHash() = %q, want %q", got, expected)
	}
}

func TestFindFuzzyMatch(t *testing.T) {
	db := setupTestDB(t)
	svc := NewImportService(db)

	// Insert test transactions
	db.MustExec(`INSERT INTO transactions (source_hash, raw_date, raw_description, raw_amount, amount, statement_id) VALUES
		('existing1', '2025-01-15', 'TESCO STORES 1234', -25.50, -25.50, 1),
		('existing2', '2025-01-15', 'UBER EATS LONDON', -18.00, -18.00, 1),
		('existing3', '2025-01-20', 'AMAZON PRIME', -9.99, -9.99, 1)`)

	// Also insert a receipt-only transaction (no statement_id)
	db.MustExec(`INSERT INTO transactions (source_hash, raw_date, raw_description, raw_amount, amount, statement_id) VALUES
		('receipt1', '2025-01-15', 'TESCO', -25.50, -25.50, NULL)`)

	t.Run("exact date and amount match found", func(t *testing.T) {
		date := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		id := svc.FindFuzzyMatch(date, -25.50, "TESCO STORES", false, false)
		if id == nil {
			t.Fatal("expected a fuzzy match, got nil")
		}
	})

	t.Run("approximate date match within 1 day", func(t *testing.T) {
		// Date is 1 day off from existing transaction
		date := time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC)
		id := svc.FindFuzzyMatch(date, -25.50, "TESCO STORES", false, false)
		if id == nil {
			t.Fatal("expected a fuzzy match for approximate date, got nil")
		}
	})

	t.Run("no match when amount differs", func(t *testing.T) {
		date := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		id := svc.FindFuzzyMatch(date, -99.99, "TESCO STORES", false, false)
		if id != nil {
			t.Errorf("expected no match for different amount, got id=%d", *id)
		}
	})

	t.Run("no match when date is far away", func(t *testing.T) {
		date := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
		id := svc.FindFuzzyMatch(date, -25.50, "TESCO STORES", false, false)
		if id != nil {
			t.Errorf("expected no match for far date, got id=%d", *id)
		}
	})

	t.Run("receiptOnly filter", func(t *testing.T) {
		date := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		id := svc.FindFuzzyMatch(date, -25.50, "TESCO STORES", true, false)
		if id == nil {
			t.Fatal("expected receipt-only match, got nil")
		}
		// Should match the receipt-only transaction (statement_id IS NULL)
	})

	t.Run("bankOnly filter", func(t *testing.T) {
		date := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		id := svc.FindFuzzyMatch(date, -25.50, "TESCO STORES", false, true)
		if id == nil {
			t.Fatal("expected bank-only match, got nil")
		}
		// Should match the bank transaction (statement_id IS NOT NULL)
	})
}

func TestImportTransactions(t *testing.T) {
	db := setupTestDB(t)

	// Need an account and statement for FK
	db.MustExec(`INSERT INTO accounts (id, name, bank, account_type, currency) VALUES ('acc1', 'Test', 'Test Bank', 'current', 'GBP')`)
	db.MustExec(`INSERT INTO statements (id, account_id, filename, file_hash, period_start, period_end) VALUES (1, 'acc1', 'test.csv', 'filehash1', '2025-01-01', '2025-01-31')`)

	db.MustExec(`INSERT INTO categories (id, name) VALUES (1, 'Groceries')`)
	db.MustExec(`INSERT INTO rules (pattern, pattern_type, category_id, priority, is_active) VALUES ('TESCO', 'contains', 1, 10, 1)`)

	svc := NewImportService(db)

	bal1 := 1000.0
	bal2 := 975.0

	parsedTxns := []parsers.ParsedTransaction{
		{Date: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), Description: "TESCO STORES", Amount: -25.00, Balance: &bal1},
		{Date: time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC), Description: "RANDOM SHOP", Amount: -15.00, Balance: &bal2},
	}

	t.Run("new transactions are inserted", func(t *testing.T) {
		result := svc.ImportTransactions("acc1", 1, parsedTxns, nil)
		if !result.Success {
			t.Fatal("import should succeed")
		}
		if result.Imported != 2 {
			t.Errorf("expected 2 imported, got %d", result.Imported)
		}
		if result.Duplicates != 0 {
			t.Errorf("expected 0 duplicates, got %d", result.Duplicates)
		}
	})

	t.Run("classification is applied", func(t *testing.T) {
		var catSource string
		err := db.Get(&catSource, `SELECT category_source FROM transactions WHERE raw_description = 'TESCO STORES'`)
		if err != nil {
			t.Fatal(err)
		}
		if catSource != "rule" {
			t.Errorf("expected category_source='rule' for TESCO, got %q", catSource)
		}
	})

	t.Run("duplicate source_hash is skipped", func(t *testing.T) {
		// Import same transactions again
		result := svc.ImportTransactions("acc1", 1, parsedTxns, nil)
		if result.Imported != 0 {
			t.Errorf("expected 0 imported on duplicate, got %d", result.Imported)
		}
		if result.Duplicates != 2 {
			t.Errorf("expected 2 duplicates, got %d", result.Duplicates)
		}
	})
}
