package parsers

import (
	"fmt"
	"time"
)

// PDFExtractFunc is a function that sends PDF bytes to Gemini and returns structured data.
type PDFExtractFunc func(content []byte) (ok bool, data map[string]interface{}, errMsg string)

// GeminiPDFParser uses Gemini AI to extract transactions from PDFs.
type GeminiPDFParser struct {
	extract PDFExtractFunc
}

func NewGeminiPDFParser(extractFn PDFExtractFunc) *GeminiPDFParser {
	return &GeminiPDFParser{extract: extractFn}
}

func (p *GeminiPDFParser) FileTypes() []string { return []string{".pdf"} }
func (p *GeminiPDFParser) BankID() string      { return "gemini-pdf" }
func (p *GeminiPDFParser) BankName() string     { return "AI-Extracted PDF" }

func (p *GeminiPDFParser) Parse(content []byte) (ParsedStatement, error) {
	ok, data, errMsg := p.extract(content)
	if !ok {
		return ParsedStatement{}, fmt.Errorf("gemini extraction failed: %s", errMsg)
	}

	result := ParsedStatement{}

	if ps, ok := data["period_start"].(string); ok {
		if t, err := time.Parse("2006-01-02", ps); err == nil {
			result.PeriodStart = t
		}
	}
	if pe, ok := data["period_end"].(string); ok {
		if t, err := time.Parse("2006-01-02", pe); err == nil {
			result.PeriodEnd = t
		}
	}
	if ob, ok := data["opening_balance"].(float64); ok {
		result.OpeningBalance = &ob
	}
	if cb, ok := data["closing_balance"].(float64); ok {
		result.ClosingBalance = &cb
	}

	txns, _ := data["transactions"].([]interface{})
	for _, raw := range txns {
		txnMap, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}

		dateStr, _ := txnMap["date"].(string)
		txnDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		desc, _ := txnMap["description"].(string)
		amount, _ := txnMap["amount"].(float64)

		var balance *float64
		if b, ok := txnMap["balance"].(float64); ok {
			balance = &b
		}

		result.Transactions = append(result.Transactions, ParsedTransaction{
			Date:        txnDate,
			Description: desc,
			Amount:      amount,
			Balance:     balance,
		})
	}

	return result, nil
}
