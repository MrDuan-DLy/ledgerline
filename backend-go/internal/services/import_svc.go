package services

import (
	"crypto/sha256"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/anthropics/accounting-tool/backend-go/internal/parsers"
)

// ImportService handles importing bank statements.
type ImportService struct {
	db       *sqlx.DB
	classify *ClassifyService
}

func NewImportService(db *sqlx.DB) *ImportService {
	return &ImportService{
		db:       db,
		classify: NewClassifyService(db),
	}
}

// ComputeFileHash returns SHA256 hex of file content.
func ComputeFileHash(content []byte) string {
	h := sha256.Sum256(content)
	return fmt.Sprintf("%x", h)
}

// ComputeTransactionHash creates a unique hash for deduplication.
func ComputeTransactionHash(txnDate time.Time, description string, amount float64, balance *float64) string {
	balStr := "null"
	if balance != nil {
		balStr = fmt.Sprintf("%.2f", *balance)
	}
	data := fmt.Sprintf("%s|%s|%.2f|%s", txnDate.Format("2006-01-02"), description, amount, balStr)
	h := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", h)
}

// FindFuzzyMatch finds an existing transaction that likely represents the same purchase.
func (s *ImportService) FindFuzzyMatch(txnDate time.Time, amount float64, description string, receiptOnly, bankOnly bool) *int {
	epsilon := 0.01
	start := txnDate.AddDate(0, 0, -1).Format("2006-01-02")
	end := txnDate.AddDate(0, 0, 1).Format("2006-01-02")
	dateStr := txnDate.Format("2006-01-02")

	q := `SELECT id, raw_date, raw_description, amount FROM transactions
	      WHERE amount >= ? AND amount <= ?
	      AND raw_date >= ? AND raw_date <= ?`
	args := []interface{}{amount - epsilon, amount + epsilon, start, end}

	if receiptOnly {
		q += " AND statement_id IS NULL"
	}
	if bankOnly {
		q += " AND statement_id IS NOT NULL"
	}

	type row struct {
		ID             int     `db:"id"`
		RawDate        string  `db:"raw_date"`
		RawDescription string  `db:"raw_description"`
		Amount         float64 `db:"amount"`
	}

	var candidates []row
	if err := s.db.Select(&candidates, q, args...); err != nil {
		log.Printf("FindFuzzyMatch query error: %v", err)
		return nil
	}

	if len(candidates) == 0 {
		return nil
	}

	descUpper := strings.ToUpper(description)
	tokens := filterTokens(strings.Fields(descUpper), 3)

	var best *row
	bestScore := 0

	for i := range candidates {
		c := &candidates[i]
		score := 0
		if c.RawDate == dateStr {
			score += 2
		} else {
			score += 1
		}

		txnDesc := strings.ToUpper(c.RawDescription)
		if len(tokens) > 0 && anyTokenIn(tokens, txnDesc) {
			score += 1
		}
		txnTokens := filterTokens(strings.Fields(txnDesc), 3)
		if len(txnTokens) > 0 && anyTokenIn(txnTokens, descUpper) {
			score += 1
		}

		if score > bestScore {
			best = c
			bestScore = score
		}
	}

	if best != nil && bestScore >= 3 {
		return &best.ID
	}
	return nil
}

// EnsureAccount creates an account if not exists and returns the ID.
func (s *ImportService) EnsureAccount(id, name, bank, accountType, currency string) string {
	var count int
	if err := s.db.Get(&count, `SELECT COUNT(*) FROM accounts WHERE id = ?`, id); err != nil {
		log.Printf("EnsureAccount check error: %v", err)
		return id
	}
	if count == 0 {
		if _, err := s.db.Exec(
			`INSERT INTO accounts (id, name, bank, account_type, currency) VALUES (?, ?, ?, ?, ?)`,
			id, name, bank, accountType, currency); err != nil {
			log.Printf("EnsureAccount insert error: %v", err)
		}
	}
	return id
}

// ImportResult holds the outcome of an import.
type ImportResult struct {
	Success     bool   `json:"success"`
	StatementID *int   `json:"statement_id"`
	Imported    int    `json:"imported"`
	Duplicates  int    `json:"duplicates"`
	Classified  int    `json:"classified"`
	Message     string `json:"message"`
}

// ImportTransactions imports parsed transactions into the database inside a transaction.
func (s *ImportService) ImportTransactions(
	accountID string,
	statementID int,
	txns []parsers.ParsedTransaction,
	categoryMap map[string]int,
) ImportResult {
	tx, err := s.db.Beginx()
	if err != nil {
		return ImportResult{Success: false, Message: "failed to begin transaction"}
	}
	defer tx.Rollback()

	imported := 0
	skipped := 0
	classified := 0

	for _, txn := range txns {
		sourceHash := ComputeTransactionHash(txn.Date, txn.Description, txn.Amount, txn.Balance)

		// Check for exact duplicate
		var count int
		if err := tx.Get(&count, `SELECT COUNT(*) FROM transactions WHERE source_hash = ?`, sourceHash); err != nil {
			log.Printf("ImportTransactions dedup check error: %v", err)
			continue
		}
		if count > 0 {
			skipped++
			continue
		}

		// Check for receipt-created transaction that matches
		fuzzyID := s.FindFuzzyMatch(txn.Date, txn.Amount, txn.Description, true, false)
		if fuzzyID != nil {
			// Absorb: upgrade receipt transaction with bank data
			var balVal interface{} = nil
			if txn.Balance != nil {
				balVal = *txn.Balance
			}

			// Check current category_source
			var catSource string
			if err := tx.Get(&catSource, `SELECT category_source FROM transactions WHERE id = ?`, *fuzzyID); err != nil {
				log.Printf("ImportTransactions catSource lookup error: %v", err)
			}

			if _, err := tx.Exec(
				`UPDATE transactions SET source_hash = ?, statement_id = ?,
				 raw_description = ?, raw_balance = ?, raw_amount = ? WHERE id = ?`,
				sourceHash, statementID, txn.Description, balVal, txn.Amount, *fuzzyID); err != nil {
				log.Printf("ImportTransactions absorb error: %v", err)
				continue
			}

			if catSource != "manual" && catSource != "receipt" {
				catID, source := s.classifyOrMap(txn, categoryMap)
				if catID != nil {
					if _, err := tx.Exec(`UPDATE transactions SET category_id = ?, category_source = ? WHERE id = ?`,
						*catID, source, *fuzzyID); err != nil {
						log.Printf("ImportTransactions classify update error: %v", err)
					}
					classified++
				}
			}

			imported++
			continue
		}

		// Create new transaction
		catID, catSource := s.classifyOrMap(txn, categoryMap)

		var balVal interface{} = nil
		if txn.Balance != nil {
			balVal = *txn.Balance
		}
		var catIDVal interface{} = nil
		if catID != nil {
			catIDVal = *catID
			classified++
		}

		var notesVal interface{} = nil
		if txn.Notes != nil {
			notesVal = *txn.Notes
		}

		if _, err := tx.Exec(
			`INSERT INTO transactions
			 (statement_id, source_hash, raw_date, raw_description, raw_amount, raw_balance,
			  amount, description, category_id, category_source, notes)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			statementID, sourceHash, txn.Date.Format("2006-01-02"), txn.Description,
			txn.Amount, balVal, txn.Amount, txn.Description, catIDVal, catSource, notesVal); err != nil {
			log.Printf("ImportTransactions insert error: %v", err)
			continue
		}
		imported++
	}

	if err := tx.Commit(); err != nil {
		return ImportResult{Success: false, Message: "failed to commit transaction"}
	}

	return ImportResult{
		Success:     true,
		StatementID: &statementID,
		Imported:    imported,
		Duplicates:  skipped,
		Classified:  classified,
		Message:     fmt.Sprintf("Imported %d transactions, skipped %d duplicates", imported, skipped),
	}
}

func (s *ImportService) classifyOrMap(txn parsers.ParsedTransaction, categoryMap map[string]int) (*int, string) {
	if txn.MappedCategory != nil && categoryMap != nil {
		if catID, ok := categoryMap[*txn.MappedCategory]; ok {
			return &catID, "merchant"
		}
	}

	catID, source := s.classify.Classify(txn.Description)
	return catID, source
}

// BuildCategoryMap loads category names to IDs.
func BuildCategoryMap(db *sqlx.DB) map[string]int {
	type row struct {
		ID   int    `db:"id"`
		Name string `db:"name"`
	}
	var rows []row
	if err := db.Select(&rows, `SELECT id, name FROM categories`); err != nil {
		log.Printf("BuildCategoryMap error: %v", err)
		return make(map[string]int)
	}
	m := make(map[string]int, len(rows))
	for _, r := range rows {
		m[r.Name] = r.ID
	}
	return m
}

// CreateStatement creates a statement record and returns its ID.
func (s *ImportService) CreateStatement(
	accountID, filename, fileHash string,
	periodStart, periodEnd time.Time,
	openingBalance, closingBalance *float64,
	rawText string,
) (int, error) {
	var obVal, cbVal interface{}
	if openingBalance != nil {
		obVal = *openingBalance
	}
	if closingBalance != nil {
		cbVal = *closingBalance
	}

	res, err := s.db.Exec(
		`INSERT INTO statements (account_id, filename, file_hash, period_start, period_end,
		  opening_balance, closing_balance, raw_text)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		accountID, filename, fileHash,
		periodStart.Format("2006-01-02"), periodEnd.Format("2006-01-02"),
		obVal, cbVal, rawText)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

// CheckDuplicateFile checks if a file with the same hash already exists.
func (s *ImportService) CheckDuplicateFile(fileHash string) bool {
	var count int
	if err := s.db.Get(&count, `SELECT COUNT(*) FROM statements WHERE file_hash = ?`, fileHash); err != nil {
		log.Printf("CheckDuplicateFile error: %v", err)
		return false
	}
	return count > 0
}
