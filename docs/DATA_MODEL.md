# Data Model

**Database**: SQLite (`data/accounting.db`)
**ORM**: SQLAlchemy
**Schemas**: Pydantic v2

## Entity Relationship Overview

```
Account 1──* Statement 1──* Transaction *──1 Category
                                  |              |
                                  |         Rule *──1 Category
                                  |
                            Receipt 1──* ReceiptItem
                                  |
                            ImportSession 1──* ImportItem

Merchant *──1 Category
Budget   *──1 Category
AuditLog (standalone)
```

## Models

### Account (`accounts`)

| Column | Type | Constraints | Default |
|--------|------|-------------|---------|
| id | String(50) | PK | - |
| name | String(100) | NOT NULL | - |
| bank | String(50) | NOT NULL | - |
| account_type | String(20) | NOT NULL | - |
| currency | String(3) | - | `'GBP'` |
| created_at | DateTime | - | `utcnow` |

### Statement (`statements`)

| Column | Type | Constraints | Default |
|--------|------|-------------|---------|
| id | Integer | PK auto | - |
| account_id | String(50) | FK(accounts.id) | - |
| filename | String(255) | NOT NULL | - |
| file_hash | String(64) | UNIQUE, NOT NULL | - |
| period_start | Date | NOT NULL | - |
| period_end | Date | NOT NULL | - |
| opening_balance | Float | - | NULL |
| closing_balance | Float | - | NULL |
| raw_text | Text | - | NULL |
| imported_at | DateTime | - | `utcnow` |

### Transaction (`transactions`)

| Column | Type | Constraints | Default | Notes |
|--------|------|-------------|---------|-------|
| id | Integer | PK auto | - | |
| statement_id | Integer | FK(statements.id) | NULL | |
| source_hash | String(64) | UNIQUE, NOT NULL | - | SHA256(date+desc+amount+balance) |
| raw_date | Date | NOT NULL | - | Immutable |
| raw_description | Text | NOT NULL | - | Immutable |
| raw_amount | Float | NOT NULL | - | Immutable; negative=expense |
| raw_balance | Float | - | NULL | Immutable |
| effective_date | Date | - | NULL | User-corrected |
| description | Text | - | NULL | User-cleaned |
| amount | Float | NOT NULL | - | Derived/corrected |
| category_id | Integer | FK(categories.id) | NULL | |
| category_source | String(20) | - | `'unclassified'` | manual/rule/merchant/unclassified |
| is_excluded | Boolean | - | False | For transfers, large one-offs |
| is_reconciled | Boolean | - | False | |
| reconciled_at | DateTime | - | NULL | |
| notes | Text | - | NULL | |
| created_at | DateTime | - | `utcnow` | |
| updated_at | DateTime | - | `utcnow` | onupdate |

### Category (`categories`)

| Column | Type | Constraints | Default |
|--------|------|-------------|---------|
| id | Integer | PK auto | - |
| name | String(50) | UNIQUE, NOT NULL | - |
| parent_id | Integer | FK(categories.id) | NULL |
| icon | String(50) | - | NULL |
| color | String(7) | - | NULL |
| is_expense | Boolean | - | True |

Self-referential: `parent` -> Category, `children` via backref.

### Rule (`rules`)

| Column | Type | Constraints | Default |
|--------|------|-------------|---------|
| id | Integer | PK auto | - |
| pattern | String(255) | NOT NULL | - |
| pattern_type | String(20) | - | `'contains'` |
| category_id | Integer | FK(categories.id), NOT NULL | - |
| priority | Integer | - | 0 |
| is_active | Boolean | - | True |
| created_from_txn_id | Integer | - | NULL |
| created_at | DateTime | - | `utcnow` |

### Receipt (`receipts`)

| Column | Type | Constraints | Default |
|--------|------|-------------|---------|
| id | Integer | PK auto | - |
| image_path | String(255) | NOT NULL | - |
| image_hash | String(64) | UNIQUE, NOT NULL | - |
| merchant_name | String(255) | - | NULL |
| receipt_date | Date | - | NULL |
| receipt_time | String(20) | - | NULL |
| total_amount | Float | - | NULL |
| currency | String(10) | - | NULL |
| payment_method | String(50) | - | NULL |
| status | String(20) | - | `'pending'` |
| ocr_raw | Text | - | NULL |
| ocr_json | Text | - | NULL |
| transaction_id | Integer | FK(transactions.id) | NULL |
| matched_transaction_id | Integer | FK(transactions.id) | NULL |
| matched_reason | String(255) | - | NULL |
| created_at | DateTime | - | `utcnow` |

### ReceiptItem (`receipt_items`)

| Column | Type | Constraints | Default |
|--------|------|-------------|---------|
| id | Integer | PK auto | - |
| receipt_id | Integer | FK(receipts.id), NOT NULL | - |
| name | String(255) | NOT NULL | - |
| quantity | Float | - | NULL |
| unit_price | Float | - | NULL |
| line_total | Float | - | NULL |

### Merchant (`merchants`)

| Column | Type | Constraints | Default |
|--------|------|-------------|---------|
| id | Integer | PK auto | - |
| name | String(255) | UNIQUE, NOT NULL | - |
| patterns | Text | - | `'[]'` |
| category_id | Integer | FK(categories.id) | NULL |
| created_at | DateTime | - | `utcnow` |

Helper methods: `get_patterns()`, `set_patterns()`, `add_pattern()` for JSON list management.

### Budget (`budgets`)

| Column | Type | Constraints | Default |
|--------|------|-------------|---------|
| id | Integer | PK auto | - |
| category_id | Integer | FK(categories.id), UNIQUE | NOT NULL |
| monthly_limit | Float | NOT NULL | - |
| created_at | DateTime | - | `utcnow` |

### ImportSession (`import_sessions`)

| Column | Type | Constraints | Default |
|--------|------|-------------|---------|
| id | String(36) | PK | UUID v4 |
| source_type | String(20) | NOT NULL | - |
| source_file | String(255) | NOT NULL | - |
| file_hash | String(64) | NOT NULL | - |
| file_path | String(512) | NOT NULL | - |
| page_count | Integer | - | NULL |
| page_image_paths | JSON | - | NULL |
| metadata_json | Text | - | NULL |
| ai_usage_json | Text | - | NULL |
| status | String(20) | - | `'pending'` |
| created_at | DateTime | - | `utcnow` |

### ImportItem (`import_items`)

| Column | Type | Constraints | Default |
|--------|------|-------------|---------|
| id | Integer | PK auto | - |
| session_id | String(36) | FK(import_sessions.id), NOT NULL | - |
| page_num | Integer | - | NULL |
| extracted_date | Date | - | NULL |
| extracted_description | Text | - | NULL |
| extracted_amount | Float | - | NULL |
| extracted_balance | Float | - | NULL |
| extracted_merchant | String(255) | - | NULL |
| extracted_items_json | Text | - | NULL |
| raw_ai_json | Text | - | NULL |
| status | String(20) | - | `'pending'` |
| duplicate_of_id | Integer | FK(transactions.id) | NULL |
| duplicate_score | Float | - | NULL |
| duplicate_reason | String(255) | - | NULL |
| created_at | DateTime | - | `utcnow` |

### AuditLog (`audit_log`)

| Column | Type | Constraints | Default |
|--------|------|-------------|---------|
| id | Integer | PK auto | - |
| table_name | String(50) | NOT NULL | - |
| record_id | Integer | NOT NULL | - |
| action | String(10) | NOT NULL | - |
| old_values | Text | - | NULL |
| new_values | Text | - | NULL |
| created_at | DateTime | - | `utcnow` |

## Seed Data (init_db.py)

**Default categories** (15): Income, Transfer In, Housing, Groceries, Eating Out, Transport, Shopping, Subscriptions, Entertainment, Health, Travel, Education, Transfer Out, Fees & Charges, Other

**Default rules** (40+): Pattern-based classification for common UK merchants

**Default merchants** (23+): Canonical names with normalized aliases (Tesco, Sainsbury's, Uber Eats, Netflix, TfL, etc.)

**Default account**: HSBC Current Account (id: `hsbc-main`)

## Design Principles

1. **Immutable raw data**: `raw_*` fields never modified after import
2. **Deduplication**: `source_hash` (transaction) and `file_hash` (statement) prevent duplicates
3. **Classification priority**: manual > rule > merchant > unclassified
4. **Audit trail**: `category_source` tracks classification origin
5. **Review before commit**: ImportSession/ImportItem pattern for AI-extracted data
