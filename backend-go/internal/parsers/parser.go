package parsers

import "time"

// ParsedTransaction holds a single transaction extracted from a statement.
type ParsedTransaction struct {
	Date           time.Time
	Description    string
	Amount         float64
	Balance        *float64
	MappedCategory *string
	Notes          *string
}

// ParsedStatement holds the result of parsing a bank statement file.
type ParsedStatement struct {
	PeriodStart    time.Time
	PeriodEnd      time.Time
	OpeningBalance *float64
	ClosingBalance *float64
	Transactions   []ParsedTransaction
	RawText        string
}

// Parser is the interface for bank statement parsers.
type Parser interface {
	Parse(content []byte) (ParsedStatement, error)
	FileTypes() []string
	BankID() string
	BankName() string
}
