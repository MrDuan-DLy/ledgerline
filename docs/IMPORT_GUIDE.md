# Import Guide

How to add support for new bank statement formats.

## Parser Interface (Go)

The Go backend defines a standard parser interface in `internal/parser/`:

```go
// ParsedTransaction represents a single transaction from a bank statement.
type ParsedTransaction struct {
    Date        time.Time
    Description string
    Amount      float64  // negative = expense, positive = income
    Balance     *float64 // running balance (nil if unavailable)
    Notes       string   // optional extra info
}

// ParsedStatement is the output of parsing a bank statement file.
type ParsedStatement struct {
    PeriodStart    time.Time
    PeriodEnd      time.Time
    OpeningBalance *float64
    ClosingBalance *float64
    Transactions   []ParsedTransaction
    RawText        string // original extracted text for audit
}

// Parser converts raw file bytes into a ParsedStatement.
type Parser interface {
    Parse(content []byte) (*ParsedStatement, error)
    // SupportedFormats returns MIME types this parser handles.
    SupportedFormats() []string
}
```

## Implementing a New Parser

1. Create a new file under `internal/parser/`, e.g. `internal/parser/barclays_csv.go`.

2. Implement the `Parser` interface:

```go
package parser

import (
    "encoding/csv"
    "bytes"
    "strconv"
    "time"
)

type BarclaysCSVParser struct{}

func (p *BarclaysCSVParser) SupportedFormats() []string {
    return []string{"text/csv"}
}

func (p *BarclaysCSVParser) Parse(content []byte) (*ParsedStatement, error) {
    reader := csv.NewReader(bytes.NewReader(content))
    records, err := reader.ReadAll()
    if err != nil {
        return nil, err
    }

    stmt := &ParsedStatement{}
    for i, row := range records {
        if i == 0 {
            continue // skip header
        }
        // Adapt column indices to your bank's CSV format
        txnDate, _ := time.Parse("02/01/2006", row[0])
        amount, _ := strconv.ParseFloat(row[3], 64)
        balance, _ := strconv.ParseFloat(row[4], 64)

        stmt.Transactions = append(stmt.Transactions, ParsedTransaction{
            Date:        txnDate,
            Description: row[1],
            Amount:      amount,
            Balance:     &balance,
        })
    }

    if len(stmt.Transactions) > 0 {
        stmt.PeriodStart = stmt.Transactions[0].Date
        stmt.PeriodEnd = stmt.Transactions[len(stmt.Transactions)-1].Date
    }

    return stmt, nil
}
```

3. Register the parser in `internal/parser/registry.go`:

```go
var Registry = map[string]Parser{
    "hsbc":     &HSBCPDFParser{},
    "barclays": &BarclaysCSVParser{},
}
```

## Supported Import Methods

### Direct Upload (PDF/CSV)

Upload bank statement files through the API or UI. The system selects the appropriate parser based on the account's bank setting.

- **PDF statements**: Parsed using bank-specific PDF parsers (e.g., HSBC UK format)
- **CSV exports**: Parsed using bank-specific CSV parsers

**API**: `POST /api/statements` with multipart form data containing the file and `account_id`.

### AI-Powered Extraction (Gemini)

For banks without a dedicated parser, or when PDF parsing fails, the system can use Google Gemini to extract transactions from any statement format.

- Sends the full PDF/image to Gemini with structured output schema
- Returns transactions in a standard format for review before import
- Requires `GEMINI_API_KEY` environment variable

**API**: `POST /api/imports/extract` with the file. Returns extracted transactions for review, then confirm via `POST /api/imports/confirm`.

### Receipt Scanning

Individual receipts (photos/images) can be scanned using Gemini OCR:

- Extracts merchant name, date, total, and line items
- Automatically matches against existing bank transactions
- Creates standalone transactions if no bank match found

**API**: `POST /api/receipts/scan` with the image file.

## Amount Convention

All amounts use the same sign convention:
- **Negative** = money out (expenses, transfers out)
- **Positive** = money in (income, transfers in)

Parsers must normalize to this convention regardless of how the bank formats the data.
