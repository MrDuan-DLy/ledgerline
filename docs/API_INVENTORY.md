# API Endpoint Inventory

**Total Endpoints**: 51
**Router Prefixes**: 8
**Static File Mounts**: 2

## Root

| Method | Path | Function | Description |
|--------|------|----------|-------------|
| GET | `/` | `root` | Status message |
| GET | `/health` | `health` | Health check |

## Transactions (`/api/transactions`)

| Method | Path | Function | Parameters | Response |
|--------|------|----------|------------|----------|
| GET | `/` | `list_transactions` | Query: `page`, `page_size`, `start_date`, `end_date`, `category_id`, `unclassified_only`, `excluded_only`, `hide_excluded`, `search`, `statement_id` | `TransactionListResponse` |
| GET | `/{transaction_id}` | `get_transaction` | Path: `transaction_id` | `TransactionResponse` |
| PATCH | `/{transaction_id}` | `update_transaction` | Path: `transaction_id`, Body: `TransactionUpdate` | `TransactionResponse` |
| DELETE | `/{transaction_id}` | `delete_transaction` | Path: `transaction_id` | `{success}` |
| POST | `/bulk-delete` | `bulk_delete` | Body: `list[int]` | `{deleted}` |
| POST | `/bulk-exclude` | `bulk_exclude` | Body: `list[int]`, Query: `exclude` | `{updated}` |
| POST | `/bulk-classify` | `bulk_classify` | Body: `list[int]`, `category_id` | `{updated}` |
| GET | `/stats/summary` | `get_summary` | Query: `start_date`, `end_date` | Summary object |
| GET | `/stats/series` | `get_series` | Query: `start_date`, `end_date` | `StatsSeriesResponse` |
| GET | `/stats/monthly` | `get_monthly_spend` | Query: `start_date`, `end_date`, `category_id` | `MonthlySpendResponse` |
| GET | `/stats/pace` | `get_spending_pace` | - | `SpendingPaceResponse` |

## Statements (`/api/statements`)

| Method | Path | Function | Parameters | Response |
|--------|------|----------|------------|----------|
| GET | `/` | `list_statements` | - | `list[StatementResponse]` |
| POST | `/upload` | `upload_statement` | File: PDF/CSV | `ImportResult` |
| GET | `/{statement_id}` | `get_statement` | Path: `statement_id` | `StatementResponse` |

## Categories (`/api/categories`)

| Method | Path | Function | Parameters | Response |
|--------|------|----------|------------|----------|
| GET | `/` | `list_categories` | - | `list[CategoryResponse]` |
| GET | `/tree` | `list_categories_tree` | - | `list[CategoryResponse]` (nested) |
| POST | `/` | `create_category` | Body: `CategoryCreate` | `CategoryResponse` |
| DELETE | `/{category_id}` | `delete_category` | Path: `category_id` | `{success}` |

## Rules (`/api/rules`)

| Method | Path | Function | Parameters | Response |
|--------|------|----------|------------|----------|
| GET | `/` | `list_rules` | - | `list[RuleResponse]` |
| POST | `/` | `create_rule` | Body: `RuleCreate` | `RuleResponse` |
| DELETE | `/{rule_id}` | `delete_rule` | Path: `rule_id` | `{success}` |
| PATCH | `/{rule_id}/toggle` | `toggle_rule` | Path: `rule_id` | `{active}` |
| POST | `/reclassify` | `reclassify_all` | - | `{updated}` |

## Receipts (`/api/receipts`)

| Method | Path | Function | Parameters | Response |
|--------|------|----------|------------|----------|
| GET | `/` | `list_receipts` | - | `list[ReceiptResponse]` |
| GET | `/{receipt_id}` | `get_receipt` | Path: `receipt_id` | `ReceiptResponse` |
| GET | `/{receipt_id}/image` | `get_receipt_image` | Path: `receipt_id` | FileResponse |
| GET | `/by-transaction/{transaction_id}` | `get_receipt_by_transaction` | Path: `transaction_id` | `ReceiptResponse` |
| POST | `/upload` | `upload_receipt` | File: image | `ReceiptUploadResult` |
| POST | `/upload-batch` | `upload_receipts_batch` | Files: images[] | `list[ReceiptUploadResult]` |
| POST | `/{receipt_id}/confirm` | `confirm_receipt` | Path: `receipt_id`, Body: `ReceiptConfirmRequest` | Transaction |
| PATCH | `/items/{item_id}` | `update_receipt_item` | Path: `item_id`, Body: `ReceiptItemUpdate` | `{success}` |

## Imports (`/api/imports`)

| Method | Path | Function | Parameters | Response |
|--------|------|----------|------------|----------|
| POST | `/upload` | `upload_for_review` | File: PDF/image | `ImportUploadResult` |
| GET | `/` | `list_sessions` | - | `list[ImportSessionResponse]` |
| GET | `/{session_id}` | `get_session` | Path: `session_id` | `ImportSessionResponse` |
| PATCH | `/{session_id}/items/{item_id}` | `update_item` | Path: `session_id`, `item_id`, Body: `ImportItemUpdate` | `ImportItemResponse` |
| POST | `/{session_id}/confirm` | `confirm_session` | Path: `session_id` | Confirmation object |
| GET | `/{session_id}/source` | `get_source_file` | Path: `session_id` | FileResponse |
| GET | `/{session_id}/pages/{page_num}` | `get_page_image` | Path: `session_id`, `page_num` | FileResponse |
| DELETE | `/{session_id}` | `delete_session` | Path: `session_id` | `{success}` |

## Merchants (`/api/merchants`)

| Method | Path | Function | Parameters | Response |
|--------|------|----------|------------|----------|
| GET | `/` | `list_merchants` | Query: `with_counts` | `list[MerchantResponse]` |
| GET | `/match` | `match_merchant` | Query: `raw_name` | `MerchantMatchResponse` |
| POST | `/` | `create_merchant` | Body: `MerchantCreate` | `MerchantResponse` |
| PATCH | `/{merchant_id}` | `update_merchant` | Path: `merchant_id`, Body: `MerchantUpdate` | `MerchantResponse` |
| DELETE | `/{merchant_id}` | `delete_merchant` | Path: `merchant_id` | `{success}` |
| POST | `/{merchant_id}/merge` | `merge_merchant` | Path: `merchant_id`, Body: `MerchantMergeRequest` | `MerchantResponse` |
| GET | `/{merchant_id}/transactions` | `get_merchant_transactions` | Path: `merchant_id`, Query: `start_date`, `end_date` | Merchant detail |
| POST | `/backfill` | `backfill_merchant_names` | - | Backfill result |

## Budgets (`/api/budgets`)

| Method | Path | Function | Parameters | Response |
|--------|------|----------|------------|----------|
| GET | `/` | `list_budgets` | - | `list[BudgetResponse]` |
| POST | `/` | `create_budget` | Body: `BudgetCreate` | `BudgetResponse` |
| PATCH | `/{budget_id}` | `update_budget` | Path: `budget_id`, Body: `BudgetUpdate` | `BudgetResponse` |
| DELETE | `/{budget_id}` | `delete_budget` | Path: `budget_id` | `{success}` |
| GET | `/status` | `budget_status` | - | `BudgetStatusResponse` |

## Static File Mounts

| Path | Type | Purpose |
|------|------|---------|
| `/files/pages/{path}` | StaticFiles | PDF page preview images |
| `/files/receipts/{path}` | StaticFiles | Receipt image files |
