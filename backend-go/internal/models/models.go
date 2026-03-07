package models

import (
	"database/sql"
	"time"
)

// ---------- Account ----------

type Account struct {
	ID          string    `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Bank        string    `db:"bank" json:"bank"`
	AccountType string    `db:"account_type" json:"account_type"`
	Currency    string    `db:"currency" json:"currency"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// ---------- Statement ----------

type Statement struct {
	ID             int             `db:"id" json:"id"`
	AccountID      string          `db:"account_id" json:"account_id"`
	Filename       string          `db:"filename" json:"filename"`
	FileHash       string          `db:"file_hash" json:"file_hash"`
	PeriodStart    string          `db:"period_start" json:"period_start"`
	PeriodEnd      string          `db:"period_end" json:"period_end"`
	OpeningBalance sql.NullFloat64 `db:"opening_balance" json:"opening_balance"`
	ClosingBalance sql.NullFloat64 `db:"closing_balance" json:"closing_balance"`
	RawText        sql.NullString  `db:"raw_text" json:"raw_text"`
	ImportedAt     time.Time       `db:"imported_at" json:"imported_at"`
}

// ---------- Category ----------

type Category struct {
	ID        int            `db:"id" json:"id"`
	Name      string         `db:"name" json:"name"`
	ParentID  sql.NullInt64  `db:"parent_id" json:"parent_id"`
	Icon      sql.NullString `db:"icon" json:"icon"`
	Color     sql.NullString `db:"color" json:"color"`
	IsExpense bool           `db:"is_expense" json:"is_expense"`
}

// ---------- Transaction ----------

type Transaction struct {
	ID             int             `db:"id" json:"id"`
	StatementID    sql.NullInt64   `db:"statement_id" json:"statement_id"`
	SourceHash     string          `db:"source_hash" json:"source_hash"`
	RawDate        string          `db:"raw_date" json:"raw_date"`
	RawDescription string          `db:"raw_description" json:"raw_description"`
	RawAmount      float64         `db:"raw_amount" json:"raw_amount"`
	RawBalance     sql.NullFloat64 `db:"raw_balance" json:"raw_balance"`
	EffectiveDate  sql.NullString  `db:"effective_date" json:"effective_date"`
	Description    sql.NullString  `db:"description" json:"description"`
	Amount         float64         `db:"amount" json:"amount"`
	CategoryID     sql.NullInt64   `db:"category_id" json:"category_id"`
	CategorySource string          `db:"category_source" json:"category_source"`
	IsExcluded     bool            `db:"is_excluded" json:"is_excluded"`
	IsReconciled   bool            `db:"is_reconciled" json:"is_reconciled"`
	ReconciledAt   sql.NullTime    `db:"reconciled_at" json:"reconciled_at"`
	Notes          sql.NullString  `db:"notes" json:"notes"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time       `db:"updated_at" json:"updated_at"`
}

// TransactionRow is used for queries that JOIN category name.
type TransactionRow struct {
	Transaction
	CategoryName sql.NullString `db:"category_name" json:"category_name"`
}

// ---------- Rule ----------

type Rule struct {
	ID               int           `db:"id" json:"id"`
	Pattern          string        `db:"pattern" json:"pattern"`
	PatternType      string        `db:"pattern_type" json:"pattern_type"`
	CategoryID       int           `db:"category_id" json:"category_id"`
	Priority         int           `db:"priority" json:"priority"`
	IsActive         bool          `db:"is_active" json:"is_active"`
	CreatedFromTxnID sql.NullInt64 `db:"created_from_txn_id" json:"created_from_txn_id"`
	CreatedAt        time.Time     `db:"created_at" json:"created_at"`
}

// RuleRow includes the joined category name.
type RuleRow struct {
	Rule
	CategoryName sql.NullString `db:"category_name" json:"category_name"`
}

// ---------- AuditLog ----------

type AuditLog struct {
	ID        int            `db:"id" json:"id"`
	TableName string         `db:"table_name" json:"table_name"`
	RecordID  int            `db:"record_id" json:"record_id"`
	Action    string         `db:"action" json:"action"`
	OldValues sql.NullString `db:"old_values" json:"old_values"`
	NewValues sql.NullString `db:"new_values" json:"new_values"`
	CreatedAt time.Time      `db:"created_at" json:"created_at"`
}

// ---------- Receipt ----------

type Receipt struct {
	ID                   int             `db:"id" json:"id"`
	ImagePath            string          `db:"image_path" json:"image_path"`
	ImageHash            string          `db:"image_hash" json:"image_hash"`
	MerchantName         sql.NullString  `db:"merchant_name" json:"merchant_name"`
	ReceiptDate          sql.NullString  `db:"receipt_date" json:"receipt_date"`
	ReceiptTime          sql.NullString  `db:"receipt_time" json:"receipt_time"`
	TotalAmount          sql.NullFloat64 `db:"total_amount" json:"total_amount"`
	Currency             sql.NullString  `db:"currency" json:"currency"`
	PaymentMethod        sql.NullString  `db:"payment_method" json:"payment_method"`
	Status               string          `db:"status" json:"status"`
	OcrRaw               sql.NullString  `db:"ocr_raw" json:"ocr_raw"`
	OcrJSON              sql.NullString  `db:"ocr_json" json:"ocr_json"`
	TransactionID        sql.NullInt64   `db:"transaction_id" json:"transaction_id"`
	MatchedTransactionID sql.NullInt64   `db:"matched_transaction_id" json:"matched_transaction_id"`
	MatchedReason        sql.NullString  `db:"matched_reason" json:"matched_reason"`
	CreatedAt            time.Time       `db:"created_at" json:"created_at"`
}

// ---------- ReceiptItem ----------

type ReceiptItem struct {
	ID        int             `db:"id" json:"id"`
	ReceiptID int             `db:"receipt_id" json:"receipt_id"`
	Name      string          `db:"name" json:"name"`
	Quantity  sql.NullFloat64 `db:"quantity" json:"quantity"`
	UnitPrice sql.NullFloat64 `db:"unit_price" json:"unit_price"`
	LineTotal sql.NullFloat64 `db:"line_total" json:"line_total"`
}

// ---------- Merchant ----------

type Merchant struct {
	ID         int           `db:"id" json:"id"`
	Name       string        `db:"name" json:"name"`
	Patterns   string        `db:"patterns" json:"patterns"`
	CategoryID sql.NullInt64 `db:"category_id" json:"category_id"`
	CreatedAt  time.Time     `db:"created_at" json:"created_at"`
}

// ---------- Budget ----------

type Budget struct {
	ID           int       `db:"id" json:"id"`
	CategoryID   int       `db:"category_id" json:"category_id"`
	MonthlyLimit float64   `db:"monthly_limit" json:"monthly_limit"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// BudgetRow includes the joined category name.
type BudgetRow struct {
	Budget
	CategoryName sql.NullString `db:"category_name" json:"category_name"`
}

// ---------- ImportSession ----------

type ImportSession struct {
	ID             string         `db:"id" json:"id"`
	SourceType     string         `db:"source_type" json:"source_type"`
	SourceFile     string         `db:"source_file" json:"source_file"`
	FileHash       string         `db:"file_hash" json:"file_hash"`
	FilePath       string         `db:"file_path" json:"file_path"`
	PageCount      sql.NullInt64  `db:"page_count" json:"page_count"`
	PageImagePaths sql.NullString `db:"page_image_paths" json:"page_image_paths"`
	MetadataJSON   sql.NullString `db:"metadata_json" json:"metadata_json"`
	AIUsageJSON    sql.NullString `db:"ai_usage_json" json:"ai_usage_json"`
	Status         string         `db:"status" json:"status"`
	CreatedAt      time.Time      `db:"created_at" json:"created_at"`
}

// ---------- ImportItem ----------

type ImportItem struct {
	ID                   int             `db:"id" json:"id"`
	SessionID            string          `db:"session_id" json:"session_id"`
	PageNum              sql.NullInt64   `db:"page_num" json:"page_num"`
	ExtractedDate        sql.NullString  `db:"extracted_date" json:"extracted_date"`
	ExtractedDescription sql.NullString  `db:"extracted_description" json:"extracted_description"`
	ExtractedAmount      sql.NullFloat64 `db:"extracted_amount" json:"extracted_amount"`
	ExtractedBalance     sql.NullFloat64 `db:"extracted_balance" json:"extracted_balance"`
	ExtractedMerchant    sql.NullString  `db:"extracted_merchant" json:"extracted_merchant"`
	ExtractedItemsJSON   sql.NullString  `db:"extracted_items_json" json:"extracted_items_json"`
	RawAIJSON            sql.NullString  `db:"raw_ai_json" json:"raw_ai_json"`
	Status               string          `db:"status" json:"status"`
	DuplicateOfID        sql.NullInt64   `db:"duplicate_of_id" json:"duplicate_of_id"`
	DuplicateScore       sql.NullFloat64 `db:"duplicate_score" json:"duplicate_score"`
	DuplicateReason      sql.NullString  `db:"duplicate_reason" json:"duplicate_reason"`
	CreatedAt            time.Time       `db:"created_at" json:"created_at"`
}

// ---------- Request / Response structs ----------

// CategoryCreate matches the Python CategoryCreate schema.
type CategoryCreate struct {
	Name      string  `json:"name"`
	ParentID  *int64  `json:"parent_id"`
	Icon      *string `json:"icon"`
	Color     *string `json:"color"`
	IsExpense *bool   `json:"is_expense"`
}

// CategoryResponse matches the Python CategoryResponse schema.
type CategoryResponse struct {
	ID        int                `json:"id"`
	Name      string             `json:"name"`
	ParentID  *int64             `json:"parent_id"`
	Icon      *string            `json:"icon"`
	Color     *string            `json:"color"`
	IsExpense bool               `json:"is_expense"`
	Children  []CategoryResponse `json:"children"`
}

// RuleCreate matches the Python RuleCreate schema.
type RuleCreate struct {
	Pattern          string `json:"pattern"`
	PatternType      string `json:"pattern_type"`
	CategoryID       int    `json:"category_id"`
	Priority         int    `json:"priority"`
	IsActive         *bool  `json:"is_active"`
	CreatedFromTxnID *int   `json:"created_from_txn_id"`
}

// RuleResponse matches the Python RuleResponse schema.
type RuleResponse struct {
	ID               int     `json:"id"`
	Pattern          string  `json:"pattern"`
	PatternType      string  `json:"pattern_type"`
	CategoryID       int     `json:"category_id"`
	Priority         int     `json:"priority"`
	IsActive         bool    `json:"is_active"`
	CategoryName     *string `json:"category_name"`
	CreatedFromTxnID *int    `json:"created_from_txn_id"`
	CreatedAt        string  `json:"created_at"`
}

// TransactionUpdate matches the Python TransactionUpdate schema.
type TransactionUpdate struct {
	EffectiveDate *string `json:"effective_date"`
	Description   *string `json:"description"`
	CategoryID    *int64  `json:"category_id"`
	Notes         *string `json:"notes"`
	IsExcluded    *bool   `json:"is_excluded"`
}

// TransactionResponse matches the Python TransactionResponse schema.
type TransactionResponse struct {
	ID             int      `json:"id"`
	StatementID    *int64   `json:"statement_id"`
	SourceHash     string   `json:"source_hash"`
	RawDate        string   `json:"raw_date"`
	RawDescription string   `json:"raw_description"`
	RawAmount      float64  `json:"raw_amount"`
	RawBalance     *float64 `json:"raw_balance"`
	EffectiveDate  *string  `json:"effective_date"`
	Description    *string  `json:"description"`
	Amount         float64  `json:"amount"`
	CategoryID     *int64   `json:"category_id"`
	CategoryName   *string  `json:"category_name"`
	CategorySource string   `json:"category_source"`
	IsExcluded     bool     `json:"is_excluded"`
	IsReconciled   bool     `json:"is_reconciled"`
	ReconciledAt   *string  `json:"reconciled_at"`
	Notes          *string  `json:"notes"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
}

// TransactionListResponse is a paginated list of transactions.
type TransactionListResponse struct {
	Items      []TransactionResponse `json:"items"`
	Total      int                   `json:"total"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
	TotalPages int                   `json:"total_pages"`
}

// BulkClassifyRequest for POST /api/transactions/bulk-classify.
type BulkClassifyRequest struct {
	TransactionIDs []int  `json:"transaction_ids"`
	CategoryID     int    `json:"category_id"`
	Source         string `json:"source"`
}

// BulkDeleteRequest for POST /api/transactions/bulk-delete.
type BulkDeleteRequest struct {
	TransactionIDs []int `json:"transaction_ids"`
}

// BulkExcludeRequest for POST /api/transactions/bulk-exclude.
type BulkExcludeRequest struct {
	TransactionIDs []int `json:"transaction_ids"`
	IsExcluded     bool  `json:"is_excluded"`
}

// StatsSummary matches the Python summary endpoint.
type StatsSummary struct {
	TotalExpenses float64 `json:"total_expenses"`
	TotalIncome   float64 `json:"total_income"`
	AvgDaily      float64 `json:"avg_daily"`
	Count         int     `json:"count"`
	Unclassified  int     `json:"unclassified"`
}

// DailySeriesPoint matches Python DailySeriesPoint.
type DailySeriesPoint struct {
	Date       string  `json:"date"`
	Net        float64 `json:"net"`
	Income     float64 `json:"income"`
	Expenses   float64 `json:"expenses"`
	Count      int     `json:"count"`
	Cumulative float64 `json:"cumulative"`
}

// CategoryTotal matches Python CategoryTotal.
type CategoryTotal struct {
	CategoryID   *int64  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Expenses     float64 `json:"expenses"`
	Income       float64 `json:"income"`
	Net          float64 `json:"net"`
	Count        int     `json:"count"`
}

// StatsSeriesResponse matches Python StatsSeriesResponse.
type StatsSeriesResponse struct {
	StartDate  *string            `json:"start_date"`
	EndDate    *string            `json:"end_date"`
	Daily      []DailySeriesPoint `json:"daily"`
	Categories []CategoryTotal    `json:"categories"`
}

// MonthlySpendPoint matches Python MonthlySpendPoint.
type MonthlySpendPoint struct {
	Month         string  `json:"month"`
	TotalExpenses float64 `json:"total_expenses"`
}

// MonthlySpendResponse matches Python MonthlySpendResponse.
type MonthlySpendResponse struct {
	StartDate  *string             `json:"start_date"`
	EndDate    *string             `json:"end_date"`
	CategoryID *int64              `json:"category_id"`
	Series     []MonthlySpendPoint `json:"series"`
}

// PaceDayPoint matches Python PaceDayPoint.
type PaceDayPoint struct {
	Day        int     `json:"day"`
	Cumulative float64 `json:"cumulative"`
}

// SpendingPaceResponse matches Python SpendingPaceResponse.
type SpendingPaceResponse struct {
	CurrentMonth  string         `json:"current_month"`
	CurrentSeries []PaceDayPoint `json:"current_series"`
	PrevMonth     string         `json:"prev_month"`
	PrevSeries    []PaceDayPoint `json:"prev_series"`
}

// BudgetCreate matches Python BudgetCreate.
type BudgetCreate struct {
	CategoryID   int     `json:"category_id"`
	MonthlyLimit float64 `json:"monthly_limit"`
}

// BudgetUpdate matches Python BudgetUpdate.
type BudgetUpdate struct {
	MonthlyLimit float64 `json:"monthly_limit"`
}

// BudgetResponse matches Python BudgetResponse.
type BudgetResponse struct {
	ID           int     `json:"id"`
	CategoryID   int     `json:"category_id"`
	CategoryName *string `json:"category_name"`
	MonthlyLimit float64 `json:"monthly_limit"`
	CreatedAt    string  `json:"created_at"`
}

// BudgetStatusItem matches Python BudgetStatusItem.
type BudgetStatusItem struct {
	CategoryID   int     `json:"category_id"`
	CategoryName string  `json:"category_name"`
	MonthlyLimit float64 `json:"monthly_limit"`
	Spent        float64 `json:"spent"`
	Remaining    float64 `json:"remaining"`
	Percent      float64 `json:"percent"`
}

// BudgetStatusResponse matches Python BudgetStatusResponse.
type BudgetStatusResponse struct {
	Month string             `json:"month"`
	Items []BudgetStatusItem `json:"items"`
}

// StatementResponse matches the Python StatementResponse.
type StatementResponse struct {
	ID             int      `json:"id"`
	AccountID      string   `json:"account_id"`
	Filename       string   `json:"filename"`
	FileHash       string   `json:"file_hash"`
	PeriodStart    string   `json:"period_start"`
	PeriodEnd      string   `json:"period_end"`
	OpeningBalance *float64 `json:"opening_balance"`
	ClosingBalance *float64 `json:"closing_balance"`
	ImportedAt     string   `json:"imported_at"`
	TxnCount       int      `json:"transaction_count"`
}

// ImportResult matches the Python ImportResult.
type ImportResult struct {
	Success     bool   `json:"success"`
	StatementID *int   `json:"statement_id"`
	Imported    int    `json:"imported"`
	Duplicates  int    `json:"duplicates"`
	Classified  int    `json:"classified"`
	Message     string `json:"message"`
}

// ReceiptItemResponse matches the Python ReceiptItemResponse.
type ReceiptItemResponse struct {
	ID        int      `json:"id"`
	Name      string   `json:"name"`
	Quantity  *float64 `json:"quantity"`
	UnitPrice *float64 `json:"unit_price"`
	LineTotal *float64 `json:"line_total"`
}

// ReceiptItemUpdate matches the Python ReceiptItemUpdate.
type ReceiptItemUpdate struct {
	Name      *string  `json:"name"`
	Quantity  *float64 `json:"quantity"`
	UnitPrice *float64 `json:"unit_price"`
	LineTotal *float64 `json:"line_total"`
}

// ReceiptResponse matches the Python ReceiptResponse.
type ReceiptResponse struct {
	ID                            int                   `json:"id"`
	ImagePath                     string                `json:"image_path"`
	ImageHash                     string                `json:"image_hash"`
	MerchantName                  *string               `json:"merchant_name"`
	ReceiptDate                   *string               `json:"receipt_date"`
	ReceiptTime                   *string               `json:"receipt_time"`
	TotalAmount                   *float64              `json:"total_amount"`
	Currency                      *string               `json:"currency"`
	PaymentMethod                 *string               `json:"payment_method"`
	Status                        string                `json:"status"`
	OcrRaw                        *string               `json:"ocr_raw"`
	OcrJSON                       *string               `json:"ocr_json"`
	TransactionID                 *int64                `json:"transaction_id"`
	MatchedTransactionID          *int64                `json:"matched_transaction_id"`
	MatchedTransactionDate        *string               `json:"matched_transaction_date"`
	MatchedTransactionAmount      *float64              `json:"matched_transaction_amount"`
	MatchedTransactionDescription *string               `json:"matched_transaction_description"`
	MatchedReason                 *string               `json:"matched_reason"`
	CreatedAt                     string                `json:"created_at"`
	Items                         []ReceiptItemResponse `json:"items"`
}

// ReceiptUploadResult matches the Python ReceiptUploadResult.
type ReceiptUploadResult struct {
	Success   bool     `json:"success"`
	ReceiptID *int     `json:"receipt_id"`
	Message   string   `json:"message"`
	Errors    []string `json:"errors"`
}

// ReceiptConfirmRequest matches the Python ReceiptConfirmRequest.
type ReceiptConfirmRequest struct {
	MerchantName  *string  `json:"merchant_name"`
	ReceiptDate   *string  `json:"receipt_date"`
	TotalAmount   *float64 `json:"total_amount"`
	Currency      *string  `json:"currency"`
	CategoryID    *int64   `json:"category_id"`
	Notes         *string  `json:"notes"`
	TransactionID *int64   `json:"transaction_id"`
}

// ImportItemUpdate matches Python ImportItemUpdate.
type ImportItemUpdate struct {
	ExtractedDate        *string  `json:"extracted_date"`
	ExtractedDescription *string  `json:"extracted_description"`
	ExtractedAmount      *float64 `json:"extracted_amount"`
	ExtractedBalance     *float64 `json:"extracted_balance"`
	ExtractedMerchant    *string  `json:"extracted_merchant"`
	Status               *string  `json:"status"`
	DuplicateOfID        *int64   `json:"duplicate_of_id"`
}

// ImportItemResponse matches Python ImportItemResponse.
type ImportItemResponse struct {
	ID                              int      `json:"id"`
	SessionID                       string   `json:"session_id"`
	PageNum                         *int     `json:"page_num"`
	ExtractedDate                   *string  `json:"extracted_date"`
	ExtractedDescription            *string  `json:"extracted_description"`
	ExtractedAmount                 *float64 `json:"extracted_amount"`
	ExtractedBalance                *float64 `json:"extracted_balance"`
	ExtractedMerchant               *string  `json:"extracted_merchant"`
	ExtractedItemsJSON              *string  `json:"extracted_items_json"`
	Status                          string   `json:"status"`
	DuplicateOfID                   *int64   `json:"duplicate_of_id"`
	DuplicateScore                  *float64 `json:"duplicate_score"`
	DuplicateReason                 *string  `json:"duplicate_reason"`
	DuplicateTransactionDate        *string  `json:"duplicate_transaction_date"`
	DuplicateTransactionDescription *string  `json:"duplicate_transaction_description"`
	DuplicateTransactionAmount      *float64 `json:"duplicate_transaction_amount"`
	CreatedAt                       string   `json:"created_at"`
}

// ImportSessionResponse matches Python ImportSessionResponse.
type ImportSessionResponse struct {
	ID             string               `json:"id"`
	SourceType     string               `json:"source_type"`
	SourceFile     string               `json:"source_file"`
	FileHash       string               `json:"file_hash"`
	PageCount      *int                 `json:"page_count"`
	PageImagePaths []string             `json:"page_image_paths"`
	MetadataJSON   *string              `json:"metadata_json"`
	AIUsageJSON    *string              `json:"ai_usage_json"`
	Status         string               `json:"status"`
	CreatedAt      string               `json:"created_at"`
	Items          []ImportItemResponse `json:"items"`
}

// ImportUploadResult matches Python ImportUploadResult.
type ImportUploadResult struct {
	Success   bool     `json:"success"`
	SessionID *string  `json:"session_id"`
	Message   string   `json:"message"`
	Errors    []string `json:"errors"`
}

// MerchantCreate matches Python MerchantCreate.
type MerchantCreate struct {
	Name       string   `json:"name"`
	Patterns   []string `json:"patterns"`
	CategoryID *int64   `json:"category_id"`
}

// MerchantUpdate matches Python MerchantUpdate.
type MerchantUpdate struct {
	Name       *string  `json:"name"`
	Patterns   []string `json:"patterns"`
	CategoryID *int64   `json:"category_id"`
}

// MerchantResponse matches Python MerchantResponse.
type MerchantResponse struct {
	ID               int      `json:"id"`
	Name             string   `json:"name"`
	Patterns         []string `json:"patterns"`
	CategoryID       *int64   `json:"category_id"`
	TransactionCount *int     `json:"transaction_count"`
	CreatedAt        string   `json:"created_at"`
}

// MerchantMatchResponse matches Python MerchantMatchResponse.
type MerchantMatchResponse struct {
	CanonicalName *string `json:"canonical_name"`
	MerchantID    *int    `json:"merchant_id"`
	Score         float64 `json:"score"`
	MatchType     string  `json:"match_type"`
}

// MerchantMergeRequest matches Python MerchantMergeRequest.
type MerchantMergeRequest struct {
	SourceMerchantID int `json:"source_merchant_id"`
}

// ErrorResponse is the standard error envelope.
type ErrorResponse struct {
	Detail string `json:"detail"`
}
