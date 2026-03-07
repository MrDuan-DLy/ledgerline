# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Ledgerline — privacy-first personal finance tracker for single-user local deployment. Import bank statements, auto-classify transactions, capture receipts with AI.

## Tech Stack

- **Backend**: Go (Chi v5 router, sqlx, SQLite via modernc.org/sqlite)
- **Frontend**: React + Vite + TypeScript
- **PDF/Receipt AI**: Google Gemini API
- **Migrations**: goose (embedded in Go binary)

## Commands

```bash
# Build
make all              # Build backend + frontend
make build            # Build Go backend only
make frontend-build   # Build frontend only

# Development
make dev              # Backend + frontend with hot reload
make test             # Run Go tests
make lint             # golangci-lint
make frontend-lint    # ESLint + tsc --noEmit

# Run
make run              # Start server (port 8000)

# Docker
make docker-build     # Build Docker image
docker compose up -d  # Run with Docker Compose
```

## Architecture

```
backend-go/
├── cmd/server/          # Entry point (main.go)
├── internal/
│   ├── config/          # Env-based configuration
│   ├── database/        # SQLite connection + goose migrations
│   ├── handlers/        # HTTP handlers (one file per resource)
│   ├── middleware/       # CORS, logging, body size limits
│   ├── models/          # Data models (structs + DB tags)
│   ├── parsers/         # Bank statement parsers (Gemini PDF, CSV)
│   └── services/        # Business logic (classify, import, merchant, receipt)
├── migrations/          # SQL migration files (goose)
└── go.mod

frontend/
├── src/
│   ├── api/client.ts    # API client
│   ├── pages/           # Dashboard, Transactions, Import, Receipts, Budgets, etc.
│   ├── components/      # Shared UI components
│   ├── contexts/        # AppConfig context (currency, locale)
│   └── utils/           # Formatting helpers
└── vite.config.ts
```

## Key Design Decisions

1. **Immutable raw data**: `raw_date`, `raw_description`, `raw_amount`, `raw_balance` are never modified after import
2. **Deduplication**: `source_hash` = SHA256(date + description + amount + balance) prevents duplicate imports
3. **Classification priority**: manual > rule > unclassified; manual classifications are never auto-overwritten
4. **Audit trail**: `category_source` tracks how each transaction was classified
5. **SQLite concurrency**: `db.SetMaxOpenConns(1)` + `_busy_timeout=5000` for safety
6. **No CGo**: Uses `modernc.org/sqlite` (pure Go) for easy cross-compilation
7. **SPA serving**: Go backend serves frontend static files + catch-all for client-side routing

## API Endpoints

- `POST /api/statements/upload` - Upload bank statement (PDF/CSV)
- `GET/PATCH /api/transactions` - List/update transactions
- `POST /api/transactions/bulk-classify` - Bulk classification
- `GET/POST /api/categories` - Category CRUD
- `GET/POST /api/rules` - Classification rules
- `POST /api/imports/upload` - Upload for AI review
- `POST /api/receipts/upload` - Upload receipt image
- `GET/POST /api/budgets` - Budget management
- `GET/POST /api/merchants` - Merchant management
- `GET /health` - Health check with DB ping

## Data

- SQLite: `data/accounting.db`
- Uploaded files: `data/uploads/`
- Config: `.env` (see `.env.example`)
