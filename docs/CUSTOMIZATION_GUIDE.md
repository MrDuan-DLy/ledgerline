# Customization Guide

How to configure the system for your use case.

## Categories

Categories are defined in `configs/examples/categories.yaml`. To customize:

1. Copy the example to your active config:
   ```bash
   cp configs/examples/categories.yaml configs/categories.yaml
   ```

2. Edit `configs/categories.yaml`:
   ```yaml
   categories:
     - name: Income
       is_expense: false
     - name: Rent
       is_expense: true
     # Add your own categories...
   ```

3. Re-run the database seed script to apply:
   ```bash
   python scripts/init_db.py
   ```

Categories are used for high-level budgeting. Keep the list short (10-20) and use merchants for detailed tracking.

## Classification Rules

Rules automatically categorize transactions by matching description text. See `configs/examples/rules_uk.yaml` for UK-specific defaults.

1. Copy and customize:
   ```bash
   cp configs/examples/rules_uk.yaml configs/rules.yaml
   ```

2. Edit `configs/rules.yaml`:
   ```yaml
   rules:
     - pattern: "WHOLE FOODS"
       pattern_type: contains
       category: Groceries
       priority: 5
   ```

**Fields:**
- `pattern`: Text to match in the transaction description
- `pattern_type`: Match strategy (`contains`, `starts_with`, `regex`)
- `category`: Target category name (must exist in categories)
- `priority`: Higher priority rules win when multiple rules match (1-100)

**Priority guidelines:**
- `1-3`: Generic/fallback rules (e.g., "BANK TRANSFER" -> Transfer Out)
- `5`: Standard rules (most rules)
- `10`: Specific overrides (e.g., "AMAZON PRIME" -> Subscriptions overrides "AMAZON" -> Shopping)

## Merchants

Merchants map messy bank descriptions to clean names. See `configs/examples/merchants_uk.yaml`.

1. Copy and customize:
   ```bash
   cp configs/examples/merchants_uk.yaml configs/merchants.yaml
   ```

2. Edit `configs/merchants.yaml`:
   ```yaml
   merchants:
     - name: Whole Foods
       patterns: ["whole foods", "whole fds mkt"]
       category: Groceries
   ```

**Fields:**
- `name`: Canonical display name for this merchant
- `patterns`: Lowercase substrings to match against transaction descriptions
- `category`: Default category (must exist in categories)

## Currency and Locale

Set via environment variables (in `.env` or `docker-compose.yml`):

| Variable | Default | Description |
|----------|---------|-------------|
| `CURRENCY_SYMBOL` | `£` | Symbol shown in the UI |
| `CURRENCY_LOCALE` | `en-GB` | Number formatting locale |

Examples:
```bash
# UK (default)
CURRENCY_SYMBOL=£
CURRENCY_LOCALE=en-GB

# US
CURRENCY_SYMBOL=$
CURRENCY_LOCALE=en-US

# EU
CURRENCY_SYMBOL=€
CURRENCY_LOCALE=de-DE
```

## Data Directory

All data is stored under `data/`:

| Path | Contents |
|------|----------|
| `data/accounting.db` | SQLite database |
| `data/uploads/` | Uploaded bank statement files |
| `data/receipts/` | Scanned receipt images |
| `data/page_images/` | Extracted PDF page images |

To change the data directory, modify `BASE_DIR` in `backend/config.py` or mount a different volume in Docker:

```yaml
# docker-compose.yml
volumes:
  - /path/to/your/data:/app/data
```

## Gemini AI (Optional)

For AI-powered statement extraction and receipt OCR:

```bash
GEMINI_API_KEY=your-api-key-here
GEMINI_MODEL=gemini-2.5-flash  # default
```

Get an API key from [Google AI Studio](https://aistudio.google.com/apikey).
