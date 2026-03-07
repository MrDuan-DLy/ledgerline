package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// GeminiUsage holds token usage and timing from a Gemini API call.
type GeminiUsage struct {
	Model            string `json:"model"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	DurationMs       int    `json:"duration_ms"`
}

// GeminiExtractService wraps the Gemini generative-AI API.
type GeminiExtractService struct {
	APIKey string
	Model  string
	client *http.Client
}

func NewGeminiExtractService(apiKey, model string) *GeminiExtractService {
	timeout := 120 * time.Second
	if v := os.Getenv("GEMINI_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			timeout = d
		}
	}
	return &GeminiExtractService{
		APIKey: apiKey,
		Model:  model,
		client: &http.Client{Timeout: timeout},
	}
}

// Structured output schemas matching Python exactly.
var pdfSchema = map[string]interface{}{
	"type": "OBJECT",
	"properties": map[string]interface{}{
		"period_start":    map[string]string{"type": "STRING"},
		"period_end":      map[string]string{"type": "STRING"},
		"opening_balance": map[string]string{"type": "NUMBER"},
		"closing_balance": map[string]string{"type": "NUMBER"},
		"transactions": map[string]interface{}{
			"type": "ARRAY",
			"items": map[string]interface{}{
				"type": "OBJECT",
				"properties": map[string]interface{}{
					"date":        map[string]string{"type": "STRING"},
					"description": map[string]string{"type": "STRING"},
					"amount":      map[string]string{"type": "NUMBER"},
					"balance":     map[string]string{"type": "NUMBER"},
					"page_number": map[string]string{"type": "INTEGER"},
				},
			},
		},
	},
}

var receiptSchema = map[string]interface{}{
	"type": "OBJECT",
	"properties": map[string]interface{}{
		"merchant_name":  map[string]string{"type": "STRING"},
		"receipt_date":   map[string]string{"type": "STRING"},
		"receipt_time":   map[string]string{"type": "STRING"},
		"total_amount":   map[string]string{"type": "NUMBER"},
		"currency":       map[string]string{"type": "STRING"},
		"payment_method": map[string]string{"type": "STRING"},
		"items": map[string]interface{}{
			"type": "ARRAY",
			"items": map[string]interface{}{
				"type": "OBJECT",
				"properties": map[string]interface{}{
					"name":       map[string]string{"type": "STRING"},
					"quantity":   map[string]string{"type": "NUMBER"},
					"unit_price": map[string]string{"type": "NUMBER"},
					"line_total": map[string]string{"type": "NUMBER"},
				},
			},
		},
	},
}

func (s *GeminiExtractService) callGemini(parts []interface{}, schema map[string]interface{}, prompt string) (bool, map[string]interface{}, string, *GeminiUsage) {
	if s.APIKey == "" {
		return false, nil, "Missing GEMINI_API_KEY", nil
	}

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent",
		s.Model)

	payload := map[string]interface{}{
		"contents": []interface{}{
			map[string]interface{}{
				"role":  "user",
				"parts": append([]interface{}{map[string]string{"text": prompt}}, parts...),
			},
		},
		"generationConfig": map[string]interface{}{
			"responseMimeType": "application/json",
			"responseSchema":   schema,
			"thinkingConfig":   map[string]interface{}{"thinkingBudget": 0},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return false, nil, "failed to marshal request", nil
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return false, nil, "failed to create request", nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", s.APIKey)

	start := time.Now()
	resp, err := s.client.Do(req)
	durationMs := int(time.Since(start).Milliseconds())

	if err != nil {
		log.Printf("Gemini API error after %dms: %v", durationMs, err)
		return false, nil, "Gemini API request failed", nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Gemini response read error after %dms: %v", durationMs, err)
		return false, nil, "failed to read Gemini response", nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Printf("Gemini returned invalid JSON after %dms", durationMs)
		return false, nil, "Invalid JSON response from Gemini", nil
	}

	// Extract token usage
	usageMeta, _ := result["usageMetadata"].(map[string]interface{})
	promptTokens := intFromMap(usageMeta, "promptTokenCount")
	completionTokens := intFromMap(usageMeta, "candidatesTokenCount")
	totalTokens := intFromMap(usageMeta, "totalTokenCount")

	usage := &GeminiUsage{
		Model:            s.Model,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		DurationMs:       durationMs,
	}

	log.Printf("Gemini [%s] %d prompt + %d completion = %d tokens in %dms",
		s.Model, promptTokens, completionTokens, totalTokens, durationMs)

	candidates, _ := result["candidates"].([]interface{})
	if len(candidates) == 0 {
		return false, nil, "No candidates in Gemini response", usage
	}

	candidate, _ := candidates[0].(map[string]interface{})
	content, _ := candidate["content"].(map[string]interface{})
	contentParts, _ := content["parts"].([]interface{})
	if len(contentParts) == 0 {
		return false, nil, "No content parts in Gemini response", usage
	}

	part, _ := contentParts[0].(map[string]interface{})
	text, _ := part["text"].(string)

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		return false, nil, fmt.Sprintf("Failed to parse structured output: %.200s", text), usage
	}

	return true, data, "", usage
}

// ExtractFromPDF sends a PDF to Gemini for structured extraction.
func (s *GeminiExtractService) ExtractFromPDF(content []byte) (bool, map[string]interface{}, string, *GeminiUsage) {
	prompt := "Extract all transactions from this bank statement PDF. " +
		"For each transaction, provide the date (YYYY-MM-DD format), " +
		"full description text, amount (negative for debits/expenses, positive for credits/income), " +
		"running balance after the transaction, and the page number it appears on. " +
		"Also extract the statement period start/end dates (YYYY-MM-DD) and opening/closing balances. " +
		"Be precise with amounts - use the exact figures shown on the statement."

	parts := []interface{}{
		map[string]interface{}{
			"inlineData": map[string]string{
				"mimeType": "application/pdf",
				"data":     base64.StdEncoding.EncodeToString(content),
			},
		},
	}

	return s.callGemini(parts, pdfSchema, prompt)
}

// ExtractFromImage sends a receipt image to Gemini for structured extraction.
func (s *GeminiExtractService) ExtractFromImage(content []byte, mimeType string) (bool, map[string]interface{}, string, *GeminiUsage) {
	prompt := "Extract receipt data from this image. " +
		"Provide the merchant name, date (YYYY-MM-DD format), time (HH:MM or empty string if not visible), " +
		"total amount (as a positive number), currency code (e.g. GBP, USD), " +
		"payment method (e.g. card, cash, or empty string if not visible), " +
		"and itemized list with name, quantity, unit price, and line total for each item."

	parts := []interface{}{
		map[string]interface{}{
			"inlineData": map[string]string{
				"mimeType": mimeType,
				"data":     base64.StdEncoding.EncodeToString(content),
			},
		},
	}

	return s.callGemini(parts, receiptSchema, prompt)
}

// DetectDuplicates checks each extracted item against existing transactions.
func DetectDuplicates(db *sqlx.DB, items []map[string]interface{}) []map[string]interface{} {
	for _, item := range items {
		itemDate, _ := item["date"].(string)
		itemAmount, ok := item["amount"].(float64)
		if itemDate == "" || !ok {
			continue
		}

		epsilon := 0.02
		type candidate struct {
			ID             int     `db:"id"`
			RawDate        string  `db:"raw_date"`
			RawDescription string  `db:"raw_description"`
			Amount         float64 `db:"amount"`
		}

		var candidates []candidate
		db.Select(&candidates,
			`SELECT id, raw_date, raw_description, amount FROM transactions
			 WHERE amount >= ? AND amount <= ?
			 AND raw_date >= date(?, '-2 days') AND raw_date <= date(?, '+2 days')`,
			itemAmount-epsilon, itemAmount+epsilon, itemDate, itemDate)

		if len(candidates) == 0 {
			continue
		}

		itemDesc := strings.ToUpper(fmt.Sprint(item["description"]))
		descTokens := filterTokens(strings.Fields(itemDesc), 3)

		var best *candidate
		bestScore := 0
		var bestReason string

		for i := range candidates {
			c := &candidates[i]
			score := 0
			var reasons []string

			if c.RawDate == itemDate {
				score += 3
				reasons = append(reasons, "exact date")
			} else {
				score += 1
				reasons = append(reasons, "approx date")
			}

			txnDesc := strings.ToUpper(c.RawDescription)
			if len(descTokens) > 0 && anyTokenIn(descTokens, txnDesc) {
				score += 2
				reasons = append(reasons, "description match")
			}

			if score > bestScore {
				best = c
				bestScore = score
				bestReason = strings.Join(reasons, ", ")
			}
		}

		if best != nil && bestScore >= 3 {
			item["duplicate_of_id"] = best.ID
			item["duplicate_score"] = float64(bestScore)
			item["duplicate_reason"] = bestReason
		}
	}
	return items
}

// ParseDate parses YYYY-MM-DD string. Returns empty on failure.
func ParseDate(value string) string {
	if value == "" {
		return ""
	}
	_, err := time.Parse("2006-01-02", value)
	if err != nil {
		return ""
	}
	return value
}

func intFromMap(m map[string]interface{}, key string) int {
	if m == nil {
		return 0
	}
	v, ok := m[key].(float64)
	if !ok {
		return 0
	}
	return int(v)
}

func filterTokens(tokens []string, minLen int) []string {
	var out []string
	for _, t := range tokens {
		if len(t) >= minLen {
			out = append(out, t)
		}
	}
	return out
}

func anyTokenIn(tokens []string, text string) bool {
	for _, t := range tokens {
		if strings.Contains(text, t) {
			return true
		}
	}
	return false
}

// Round2 rounds to 2 decimal places.
func Round2(v float64) float64 {
	return math.Round(v*100) / 100
}
