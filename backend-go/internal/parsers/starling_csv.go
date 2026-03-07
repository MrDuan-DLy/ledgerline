package parsers

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
)

// StarlingCSVParser parses Starling Bank CSV exports.
//
// CSV Format:
// Date,Counter Party,Reference,Type,Amount (GBP),Balance (GBP),Spending Category,Notes
type StarlingCSVParser struct{}

// Starling category -> our category name
var starlingCategoryMap = map[string]string{
	"GROCERIES":     "Groceries",
	"EATING_OUT":    "Food & Dining",
	"ENTERTAINMENT": "Entertainment",
	"TRANSPORT":     "Transport",
	"SHOPPING":      "Shopping",
	"BILLS":         "Utilities",
	"INCOME":        "Income",
	"TRANSFERS":     "Transfer In",
	"GENERAL":       "Other",
	"LIFESTYLE":     "Personal Care",
	"HOLIDAYS":      "Travel",
	"FAMILY":        "Other",
	"CHARITY":       "Other",
	"GAMBLING":      "Entertainment",
	"SAVINGS":       "Transfer Out",
	"PAYMENTS":      "Transfer Out",
}

func (p *StarlingCSVParser) FileTypes() []string { return []string{".csv"} }
func (p *StarlingCSVParser) BankID() string      { return "starling" }
func (p *StarlingCSVParser) BankName() string     { return "Starling Bank" }

func (p *StarlingCSVParser) Parse(content []byte) (ParsedStatement, error) {
	// Strip BOM
	text := string(content)
	text = strings.TrimPrefix(text, "\xef\xbb\xbf")

	r := csv.NewReader(strings.NewReader(text))
	header, err := r.Read()
	if err != nil {
		return ParsedStatement{}, fmt.Errorf("read header: %w", err)
	}

	colIdx := map[string]int{}
	for i, h := range header {
		colIdx[strings.TrimSpace(h)] = i
	}

	type indexedTxn struct {
		idx int
		txn ParsedTransaction
	}

	var txns []indexedTxn
	var dates []time.Time
	idx := 0

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		dateStr := getCol(row, colIdx, "Date")
		txnDate, err := time.Parse("02/01/2006", dateStr)
		if err != nil {
			continue
		}
		dates = append(dates, txnDate)

		amountStr := strings.ReplaceAll(getCol(row, colIdx, "Amount (GBP)"), ",", "")
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			continue
		}

		balanceStr := strings.ReplaceAll(getCol(row, colIdx, "Balance (GBP)"), ",", "")
		balance, err := strconv.ParseFloat(balanceStr, 64)
		if err != nil {
			continue
		}

		counterParty := strings.TrimSpace(getCol(row, colIdx, "Counter Party"))
		reference := strings.TrimSpace(getCol(row, colIdx, "Reference"))
		txnType := strings.TrimSpace(getCol(row, colIdx, "Type"))

		var description string
		if reference != "" && reference != counterParty {
			description = counterParty + " - " + reference
		} else {
			description = counterParty
		}
		if txnType != "" {
			description = description + " (" + txnType + ")"
		}

		starlingCat := strings.TrimSpace(getCol(row, colIdx, "Spending Category"))
		var mappedCat *string
		if mc, ok := starlingCategoryMap[starlingCat]; ok {
			mappedCat = &mc
		}

		notes := strings.TrimSpace(getCol(row, colIdx, "Notes"))
		var notesPtr *string
		if notes != "" {
			notesPtr = &notes
		}

		bal := balance
		txns = append(txns, indexedTxn{
			idx: idx,
			txn: ParsedTransaction{
				Date:           txnDate,
				Description:    description,
				Amount:         amount,
				Balance:        &bal,
				MappedCategory: mappedCat,
				Notes:          notesPtr,
			},
		})
		idx++
	}

	// Sort by date, preserving order within same day
	sort.SliceStable(txns, func(i, j int) bool {
		if txns[i].txn.Date.Equal(txns[j].txn.Date) {
			return txns[i].idx < txns[j].idx
		}
		return txns[i].txn.Date.Before(txns[j].txn.Date)
	})

	result := ParsedStatement{RawText: text}

	if len(dates) > 0 {
		minDate, maxDate := dates[0], dates[0]
		for _, d := range dates[1:] {
			if d.Before(minDate) {
				minDate = d
			}
			if d.After(maxDate) {
				maxDate = d
			}
		}
		result.PeriodStart = minDate
		result.PeriodEnd = maxDate
	} else {
		now := time.Now()
		result.PeriodStart = now
		result.PeriodEnd = now
	}

	sorted := make([]ParsedTransaction, len(txns))
	for i, t := range txns {
		sorted[i] = t.txn
	}

	if len(sorted) > 0 {
		opening := *sorted[0].Balance - sorted[0].Amount
		result.OpeningBalance = &opening
		closing := *sorted[len(sorted)-1].Balance
		result.ClosingBalance = &closing
	}

	result.Transactions = sorted
	return result, nil
}

func getCol(row []string, colIdx map[string]int, name string) string {
	if i, ok := colIdx[name]; ok && i < len(row) {
		return row[i]
	}
	return ""
}
