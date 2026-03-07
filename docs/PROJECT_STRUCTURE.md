# Project Structure

```
accounting-tool-production/
├── CLAUDE.md                          # AI assistant instructions
├── PLAN.md                            # Development roadmap
├── run.sh                             # Startup script (backend + frontend)
│
├── backend/
│   ├── __init__.py
│   ├── main.py                        # FastAPI app, CORS, router registration, static mounts
│   ├── database.py                    # SQLite engine, session factory, Base, migration helpers
│   ├── config.py                      # Paths (DATA_DIR, UPLOADS_DIR, RECEIPTS_DIR, PAGE_IMAGES_DIR)
│   │
│   ├── models/                        # SQLAlchemy ORM models
│   │   ├── __init__.py                # Barrel export of all models
│   │   ├── account.py                 # Account (bank accounts)
│   │   ├── audit.py                   # AuditLog (change tracking)
│   │   ├── budget.py                  # Budget (monthly spending limits)
│   │   ├── category.py               # Category (hierarchical classification)
│   │   ├── import_session.py          # ImportSession + ImportItem (AI review flow)
│   │   ├── merchant.py               # Merchant (canonical names + patterns)
│   │   ├── receipt.py                 # Receipt (OCR-extracted receipt data)
│   │   ├── receipt_item.py            # ReceiptItem (receipt line items)
│   │   ├── rule.py                    # Rule (auto-classification rules)
│   │   ├── statement.py              # Statement (imported bank statement files)
│   │   └── transaction.py            # Transaction (core financial record)
│   │
│   ├── schemas/                       # Pydantic request/response models
│   │   ├── __init__.py
│   │   ├── budget.py
│   │   ├── category.py
│   │   ├── import_session.py
│   │   ├── merchant.py
│   │   ├── receipt.py
│   │   ├── rule.py
│   │   ├── statement.py
│   │   └── transaction.py            # Includes stats schemas (series, pace, monthly)
│   │
│   ├── routers/                       # FastAPI endpoint handlers
│   │   ├── __init__.py
│   │   ├── budgets.py                 # /api/budgets
│   │   ├── categories.py             # /api/categories
│   │   ├── imports.py                 # /api/imports (AI review flow)
│   │   ├── merchants.py              # /api/merchants
│   │   ├── receipts.py               # /api/receipts
│   │   ├── rules.py                  # /api/rules
│   │   ├── statements.py             # /api/statements
│   │   └── transactions.py           # /api/transactions + /api/transactions/stats/*
│   │
│   ├── services/                      # Business logic layer
│   │   ├── __init__.py
│   │   ├── classify_service.py        # Rule-based transaction classification
│   │   ├── gemini_service.py          # Google Gemini API integration (OCR + PDF extraction)
│   │   ├── import_service.py          # Statement import pipeline (parse, dedupe, classify)
│   │   ├── merchant_service.py        # Merchant name matching (exact, token, fuzzy)
│   │   └── receipt_service.py         # Receipt upload, OCR, transaction matching
│   │
│   └── parsers/                       # Bank statement format parsers
│       ├── __init__.py
│       ├── hsbc_pdf.py                # HSBC PDF statement parser (pdfplumber)
│       └── starling_csv.py            # Starling Bank CSV parser
│
├── frontend/
│   ├── package.json                   # Dependencies: React, Recharts, TailwindCSS v4
│   ├── vite.config.ts                 # Vite config with API proxy to :8000
│   ├── postcss.config.js
│   ├── tsconfig.json
│   ├── tsconfig.app.json
│   ├── tsconfig.node.json
│   ├── eslint.config.js
│   │
│   └── src/
│       ├── main.tsx                   # React entry point
│       ├── App.tsx                    # Router, navigation shell ("Ledgerline" brand)
│       ├── index.css                  # TailwindCSS + custom styles
│       ├── vite-env.d.ts
│       │
│       ├── api/
│       │   └── client.ts             # API client (40+ methods, typed interfaces)
│       │
│       ├── pages/
│       │   ├── Dashboard.tsx          # KPIs, charts, budget tracker, recent activity
│       │   ├── Transactions.tsx       # Paginated table, search, bulk ops, inline edit
│       │   ├── Import.tsx             # File upload, CSV import, PDF review routing
│       │   ├── Review.tsx             # AI extraction review (split view: source + items)
│       │   ├── Receipts.tsx           # Batch upload, OCR results, transaction matching
│       │   ├── Categories.tsx         # Category tree CRUD
│       │   └── Budgets.tsx            # Budget CRUD with progress tracking
│       │
│       ├── components/
│       │   ├── Toast.tsx              # Toast notification system
│       │   ├── DateRangePicker.tsx    # Preset + custom date range selector
│       │   ├── ChartTooltip.tsx       # Shared Recharts tooltip
│       │   └── charts/
│       │       ├── BalanceTrajectory.tsx   # Current vs previous month pace
│       │       ├── CashflowRhythm.tsx     # Daily expense bars with average line
│       │       ├── MonthlySpend.tsx        # Monthly bars with trend line
│       │       ├── ExpenseMix.tsx          # Donut chart (top categories)
│       │       ├── KpiSparkline.tsx        # Mini sparkline for KPI cards
│       │       └── MerchantDrilldown.tsx   # Merchant-level spending detail
│       │
│       └── utils/
│           └── format.ts             # formatExpense, formatDate, formatCurrency
│
├── scripts/
│   └── init_db.py                    # DB init, seed categories/rules/merchants/account
│
├── accounting-mcp/                   # MCP server (separate module)
│   ├── server.py
│   └── README.md
│
└── data/                             # Runtime data (gitignored)
    ├── accounting.db                 # SQLite database
    ├── uploads/                      # Uploaded PDFs/CSVs
    ├── receipts/                     # Receipt images
    └── page_images/                  # Rendered PDF page PNGs
```

## Tech Stack

| Layer | Technology | Version |
|-------|-----------|---------|
| Backend | FastAPI | - |
| ORM | SQLAlchemy | - |
| Database | SQLite | - |
| PDF Parsing | pdfplumber | - |
| AI/OCR | Google Gemini API | gemini-3-flash-preview |
| Frontend | React | 18+ |
| Build Tool | Vite | - |
| Styling | TailwindCSS | v4 |
| Charts | Recharts | - |
| Python | 3.11 | conda: accounting-tool |

## Key File Counts

| Category | Count |
|----------|-------|
| Backend Python files | 28 |
| Frontend TypeScript/TSX files | 18 |
| ORM Models | 11 |
| Pydantic Schemas | 8 modules |
| API Router modules | 8 |
| Service modules | 5 |
| Parser modules | 2 |
| Frontend Pages | 7 |
| Frontend Components | 8 |
