# Personal Accounting MCP Server

Read-only MCP server for querying a personal expense tracking SQLite database.

## Setup

```bash
pip install -r requirements.txt
```

Ensure `accounting.db` is in the same directory as `server.py`.

## Running

```bash
# stdio transport (for MCP clients like Claude Desktop, OpenClaw)
python server.py

# Interactive testing with MCP Inspector
mcp dev server.py
```

## Tools

| Tool | Description |
|------|-------------|
| `list_categories` | List all expense/income categories |
| `get_account_overview` | Account info, statement coverage, transaction date range |
| `get_summary` | Total spending/income for a date range |
| `get_category_breakdown` | Top spending categories with percentages |
| `get_monthly_trend` | Monthly spending over time, optionally by category |
| `get_budget_status` | Budget limits vs actual spending for a month |
| `search_transactions` | Find transactions by text, date, category |
| `get_merchant_spend` | Spending history at a specific merchant |

## Database Schema

### transactions
Core table. Each row is a bank transaction.
- `amount`: negative = expense, positive = income
- `category_source`: manual / rule / merchant / unclassified
- `is_excluded`: True for inter-account transfers

### categories
Expense/income categories (e.g., Groceries, Dining Out).

### merchants
Known merchants with pattern aliases for matching.

### budgets
Monthly spending limits per category.

### accounts
Bank accounts (e.g., HSBC current account).

### statements
Imported bank statement files with period and balance info.

## Date Handling

All date parameters use ISO format (`YYYY-MM-DD`). The server uses `effective_date` when available, falling back to `raw_date`.

## Amount Conventions

- **Negative** = expense (money out)
- **Positive** = income/refund (money in)
