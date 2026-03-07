package services

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/anthropics/accounting-tool/backend-go/internal/models"
)

// ReceiptService handles receipt OCR and confirmation.
type ReceiptService struct {
	db        *sqlx.DB
	gemini    *GeminiExtractService
	importSvc *ImportService
	merchant  *MerchantService
}

func NewReceiptService(db *sqlx.DB, gemini *GeminiExtractService) *ReceiptService {
	return &ReceiptService{
		db:        db,
		gemini:    gemini,
		importSvc: NewImportService(db),
		merchant:  NewMerchantService(db),
	}
}

func ComputeImageHash(content []byte) string {
	h := sha256.Sum256(content)
	return fmt.Sprintf("%x", h)
}

func (s *ReceiptService) findMatch(receiptDate, totalAmount, merchantName string) (*int, *string) {
	if receiptDate == "" || totalAmount == "" {
		return nil, nil
	}

	var total float64
	fmt.Sscanf(totalAmount, "%f", &total)
	amount := -math.Abs(total)
	epsilon := 0.01

	type candidate struct {
		ID             int     `db:"id"`
		RawDate        string  `db:"raw_date"`
		RawDescription string  `db:"raw_description"`
		Amount         float64 `db:"amount"`
	}

	var candidates []candidate
	if err := s.db.Select(&candidates,
		`SELECT id, raw_date, raw_description, amount FROM transactions
		 WHERE amount >= ? AND amount <= ?
		 AND raw_date >= date(?, '-1 day') AND raw_date <= date(?, '+1 day')`,
		amount-epsilon, amount+epsilon, receiptDate, receiptDate); err != nil {
		log.Printf("findMatch query error: %v", err)
		return nil, nil
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	merchantUpper := strings.ToUpper(merchantName)
	tokens := filterTokens(strings.Fields(merchantUpper), 3)

	var best *candidate
	bestScore := -1
	var reason string

	for i := range candidates {
		c := &candidates[i]
		score := 0
		if c.RawDate == receiptDate {
			score += 2
		} else {
			score += 1
		}

		descUpper := strings.ToUpper(c.RawDescription)
		if len(tokens) > 0 && anyTokenIn(tokens, descUpper) {
			score += 1
		}

		if score > bestScore {
			best = c
			bestScore = score
			reason = "Amount match within 1 day"
			if score >= 3 {
				reason = "Amount + date + merchant match"
			}
		}
	}

	if best != nil {
		return &best.ID, &reason
	}
	return nil, nil
}

// ParseReceipt parses receipt image via Gemini and updates the receipt record.
func (s *ReceiptService) ParseReceipt(receiptID int, content []byte, mimeType string) models.ReceiptUploadResult {
	ok, data, errMsg, _ := s.gemini.ExtractFromImage(content, mimeType)
	if !ok || data == nil {
		if _, err := s.db.Exec(`UPDATE receipts SET status = 'failed' WHERE id = ?`, receiptID); err != nil {
			log.Printf("ParseReceipt status update error: %v", err)
		}
		msg := "OCR request failed"
		errs := []string{errMsg}
		if errMsg == "" {
			errs = []string{"Unknown error"}
		}
		return models.ReceiptUploadResult{
			Success:   false,
			ReceiptID: &receiptID,
			Message:   msg,
			Errors:    errs,
		}
	}

	ocrJSON, _ := json.Marshal(data)
	rawMerchant, _ := data["merchant_name"].(string)
	resolvedMerchant := s.merchant.Resolve(rawMerchant)
	receiptDate := ParseDate(strOrEmpty(data["receipt_date"]))
	receiptTime := strOrEmpty(data["receipt_time"])
	totalAmount := floatOrNil(data["total_amount"])
	currency := strOrEmpty(data["currency"])
	paymentMethod := strOrEmpty(data["payment_method"])

	matchedID, matchedReason := s.findMatch(receiptDate, fmt.Sprintf("%v", data["total_amount"]), resolvedMerchant)

	if _, err := s.db.Exec(
		`UPDATE receipts SET ocr_json = ?, merchant_name = ?, receipt_date = ?,
		 receipt_time = ?, total_amount = ?, currency = ?, payment_method = ?,
		 status = 'pending', matched_transaction_id = ?, matched_reason = ?
		 WHERE id = ?`,
		string(ocrJSON), resolvedMerchant,
		nilIfEmpty(receiptDate), nilIfEmpty(receiptTime),
		totalAmount, nilIfEmpty(currency), nilIfEmpty(paymentMethod),
		matchedID, matchedReason, receiptID); err != nil {
		log.Printf("ParseReceipt update error: %v", err)
	}

	// Delete old items, insert new
	if _, err := s.db.Exec(`DELETE FROM receipt_items WHERE receipt_id = ?`, receiptID); err != nil {
		log.Printf("ParseReceipt delete items error: %v", err)
	}
	items, _ := data["items"].([]interface{})
	for _, raw := range items {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		name := strings.TrimSpace(strOrEmpty(item["name"]))
		if name == "" {
			continue
		}
		if _, err := s.db.Exec(
			`INSERT INTO receipt_items (receipt_id, name, quantity, unit_price, line_total) VALUES (?, ?, ?, ?, ?)`,
			receiptID, name, floatOrNil(item["quantity"]), floatOrNil(item["unit_price"]), floatOrNil(item["line_total"])); err != nil {
			log.Printf("ParseReceipt insert item error: %v", err)
		}
	}

	return models.ReceiptUploadResult{
		Success:   true,
		ReceiptID: &receiptID,
		Message:   "Receipt parsed",
		Errors:    []string{},
	}
}

// ConfirmReceipt creates a transaction from a receipt.
func (s *ReceiptService) ConfirmReceipt(receiptID int, req models.ReceiptConfirmRequest) (int, error) {
	// Check if already confirmed
	var status string
	var existingTxnID sql.NullInt64
	if err := s.db.QueryRow(`SELECT status, transaction_id FROM receipts WHERE id = ?`, receiptID).Scan(&status, &existingTxnID); err != nil {
		return 0, fmt.Errorf("receipt not found: %w", err)
	}
	if status == "confirmed" && existingTxnID.Valid {
		return int(existingTxnID.Int64), nil
	}

	// If linking to existing transaction
	if req.TransactionID != nil {
		var count int
		if err := s.db.Get(&count, `SELECT COUNT(*) FROM transactions WHERE id = ?`, *req.TransactionID); err != nil {
			return 0, fmt.Errorf("transaction lookup error: %w", err)
		}
		if count == 0 {
			return 0, fmt.Errorf("matched transaction not found")
		}
		if _, err := s.db.Exec(`UPDATE receipts SET status = 'confirmed', transaction_id = ? WHERE id = ?`,
			*req.TransactionID, receiptID); err != nil {
			return 0, fmt.Errorf("receipt update error: %w", err)
		}
		return int(*req.TransactionID), nil
	}

	// Get receipt data
	var merchantName, receiptDate, currency sql.NullString
	var totalAmount sql.NullFloat64
	var ocrJSON sql.NullString
	if err := s.db.QueryRow(
		`SELECT merchant_name, receipt_date, total_amount, currency, ocr_json FROM receipts WHERE id = ?`,
		receiptID).Scan(&merchantName, &receiptDate, &totalAmount, &currency, &ocrJSON); err != nil {
		return 0, fmt.Errorf("receipt data lookup error: %w", err)
	}

	merchant := strPtrOrDefault(req.MerchantName, merchantName)
	rDate := strPtrOrDefault(req.ReceiptDate, receiptDate)
	if rDate == "" {
		rDate = time.Now().Format("2006-01-02")
	}
	total := floatPtrOrDefault(req.TotalAmount, totalAmount)
	curr := strPtrOrDefault(req.Currency, currency)
	if curr == "" {
		curr = "GBP"
	}

	if total == 0 || merchant == "" {
		return 0, fmt.Errorf("receipt missing required fields")
	}

	amount := -math.Abs(total)
	rDateParsed, _ := time.Parse("2006-01-02", rDate)
	sourceHash := ComputeTransactionHash(rDateParsed, merchant, amount, nil)

	// Check dedup
	var existCount int
	if err := s.db.Get(&existCount, `SELECT COUNT(*) FROM transactions WHERE source_hash = ?`, sourceHash); err != nil {
		log.Printf("ConfirmReceipt dedup check error: %v", err)
	}
	if existCount > 0 {
		var existID int
		if err := s.db.Get(&existID, `SELECT id FROM transactions WHERE source_hash = ?`, sourceHash); err != nil {
			return 0, fmt.Errorf("dedup lookup error: %w", err)
		}
		if _, err := s.db.Exec(`UPDATE receipts SET status = 'confirmed', transaction_id = ? WHERE id = ?`, existID, receiptID); err != nil {
			log.Printf("ConfirmReceipt receipt update error: %v", err)
		}
		return existID, nil
	}

	// Fuzzy match against bank transactions
	fuzzyID := s.importSvc.FindFuzzyMatch(rDateParsed, amount, merchant, false, true)
	if fuzzyID != nil {
		if _, err := s.db.Exec(`UPDATE receipts SET status = 'confirmed', transaction_id = ? WHERE id = ?`, *fuzzyID, receiptID); err != nil {
			log.Printf("ConfirmReceipt fuzzy update error: %v", err)
		}
		// Learn merchant pattern
		var rawOCRMerchant string
		if ocrJSON.Valid {
			var ocrData map[string]interface{}
			json.Unmarshal([]byte(ocrJSON.String), &ocrData)
			rawOCRMerchant, _ = ocrData["merchant_name"].(string)
		}
		s.merchant.GetOrCreate(merchant, rawOCRMerchant)
		return *fuzzyID, nil
	}

	// Create new transaction
	catSource := "unclassified"
	var catIDVal interface{} = nil
	if req.CategoryID != nil {
		catIDVal = *req.CategoryID
		catSource = "receipt"
	}
	var notesVal interface{} = nil
	if req.Notes != nil {
		notesVal = *req.Notes
	}

	res, err := s.db.Exec(
		`INSERT INTO transactions
		 (source_hash, raw_date, raw_description, raw_amount, amount, description, notes, category_id, category_source)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sourceHash, rDate, merchant, amount, amount, merchant, notesVal, catIDVal, catSource)
	if err != nil {
		return 0, err
	}
	txnID64, _ := res.LastInsertId()
	txnID := int(txnID64)

	if _, err := s.db.Exec(`UPDATE receipts SET status = 'confirmed', transaction_id = ? WHERE id = ?`, txnID, receiptID); err != nil {
		log.Printf("ConfirmReceipt final update error: %v", err)
	}

	// Learn merchant pattern
	var rawOCRMerchant string
	if ocrJSON.Valid {
		var ocrData map[string]interface{}
		json.Unmarshal([]byte(ocrJSON.String), &ocrData)
		rawOCRMerchant, _ = ocrData["merchant_name"].(string)
	}
	s.merchant.GetOrCreate(merchant, rawOCRMerchant)

	return txnID, nil
}

// Helpers
func strOrEmpty(v interface{}) string {
	s, _ := v.(string)
	return s
}

func floatOrNil(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	f, ok := v.(float64)
	if !ok {
		return nil
	}
	return f
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func strPtrOrDefault(ptr *string, ns sql.NullString) string {
	if ptr != nil {
		return *ptr
	}
	if ns.Valid {
		return ns.String
	}
	return ""
}

func floatPtrOrDefault(ptr *float64, nf sql.NullFloat64) float64 {
	if ptr != nil {
		return *ptr
	}
	if nf.Valid {
		return nf.Float64
	}
	return 0
}
