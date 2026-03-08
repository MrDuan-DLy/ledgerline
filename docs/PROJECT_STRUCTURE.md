# Project Structure

```
ledgerline/
├── CLAUDE.md                          # AI assistant instructions
├── Dockerfile                         # Multi-stage Docker build
├── docker-compose.yml
├── Makefile
│
├── backend-go/
│   ├── cmd/server/
│   │   └── main.go                    # Entry point, router setup, SPA serving
│   ├── internal/
│   │   ├── config/config.go           # Env-based configuration
│   │   ├── database/database.go       # SQLite connection, goose migrations
│   │   ├── handlers/                  # HTTP handlers (one per resource)
│   │   │   ├── budgets.go
│   │   │   ├── categories.go
│   │   │   ├── imports.go             # AI review flow
│   │   │   ├── merchants.go
│   │   │   ├── receipts.go
│   │   │   ├── rules.go
│   │   │   ├── statements.go          # PDF/CSV upload + parsing
│   │   │   └── transactions.go        # CRUD + stats endpoints
│   │   ├── middleware/                 # CORS, logging, body size limits
│   │   ├── models/models.go           # Data structs with DB tags
│   │   ├── parsers/                   # Bank statement parsers
│   │   │   ├── gemini_pdf.go          # AI-powered PDF extraction
│   │   │   ├── starling_csv.go        # CSV parser
│   │   │   └── parser.go             # Parser interface
│   │   └── services/                  # Business logic
│   │       ├── classify.go            # Rule-based classification (cached)
│   │       ├── gemini.go              # Gemini API client
│   │       ├── import_svc.go          # Import pipeline (parse, dedupe, classify)
│   │       ├── merchant.go            # Merchant matching (exact, token, fuzzy)
│   │       └── receipt.go             # Receipt OCR + transaction matching
│   ├── migrations/                    # SQL migration files (goose)
│   ├── go.mod
│   └── go.sum
│
├── frontend/
│   ├── index.html
│   ├── package.json
│   ├── vite.config.ts
│   ├── eslint.config.js
│   └── src/
│       ├── main.tsx                   # React entry point
│       ├── App.tsx                    # Router + navigation shell
│       ├── index.css                  # Styles
│       ├── api/client.ts             # API client (typed interfaces)
│       ├── pages/
│       │   ├── Dashboard.tsx          # KPIs, charts, budget tracker
│       │   ├── Transactions.tsx       # Paginated table, search, inline edit
│       │   ├── Import.tsx             # File upload, CSV/PDF import
│       │   ├── Review.tsx             # AI extraction review
│       │   ├── Receipts.tsx           # Batch upload, OCR, matching
│       │   ├── Categories.tsx         # Category tree CRUD
│       │   └── Budgets.tsx            # Budget CRUD with progress
│       ├── components/                # Shared UI + chart components
│       ├── contexts/AppConfig.tsx     # Currency/locale configuration
│       └── utils/format.ts           # Formatting helpers
│
├── configs/examples/                  # YAML config templates
├── docs/                              # Documentation
├── .github/workflows/                 # CI + Release pipelines
│
└── data/                              # Runtime data (gitignored)
    ├── accounting.db                  # SQLite database
    ├── uploads/                       # Uploaded PDFs/CSVs
    ├── receipts/                      # Receipt images
    └── page_images/                   # Rendered PDF page PNGs
```

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go (Chi v5 router, sqlx) |
| Database | SQLite (modernc.org/sqlite, pure Go) |
| Migrations | goose (embedded) |
| AI/OCR | Google Gemini API |
| Frontend | React 18 + TypeScript |
| Build Tool | Vite |
| Charts | Recharts |
