# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Personal finance system MVP for single-user local deployment. Uses bank statement PDFs as the source of truth.

## Tech Stack

- **Backend**: FastAPI + SQLAlchemy + SQLite
- **Frontend**: React + Vite + TailwindCSS
- **PDF Parsing**: pdfplumber
- **Python**: 3.11 (conda environment: `accounting-tool`)

## Commands

```bash
# Backend
conda activate accounting-tool
python scripts/init_db.py          # Initialize database
uvicorn backend.main:app --reload  # Development server (port 8000)

# Frontend
cd frontend
npm install
npm run dev                        # Development server (port 5173)
npm run build                      # Production build
```

## Architecture

```
backend/
├── main.py              # FastAPI app, CORS, routers
├── database.py          # SQLite engine, session, Base
├── config.py            # Paths configuration
├── models/              # SQLAlchemy ORM
├── schemas/             # Pydantic models
├── services/            # Business logic (ClassifyService, ImportService)
├── routers/             # API endpoints
└── parsers/             # Bank PDF parsers (HSBCPDFParser)

frontend/
├── src/
│   ├── api/client.ts    # API client
│   ├── pages/           # Transactions, Import
│   └── App.tsx          # Router + Navigation
└── tailwind.config.js
```

## Key Design Decisions

1. **Immutable raw data**: `raw_date`, `raw_description`, `raw_amount`, `raw_balance` are never modified after import
2. **Deduplication**: `source_hash` = SHA256(date + description + amount + balance) prevents duplicate imports
3. **Classification priority**: manual > rule > unclassified; manual classifications are never auto-overwritten
4. **Audit trail**: `category_source` tracks how each transaction was classified

## API Endpoints

- `GET/POST /api/statements` - List/upload bank statements
- `GET/PATCH /api/transactions` - List/update transactions
- `POST /api/transactions/bulk-classify` - Bulk classification
- `GET/POST /api/categories` - Category CRUD
- `GET/POST /api/rules` - Classification rules CRUD
- `POST /api/rules/reclassify` - Re-run rules on all non-manual transactions

## Data

- SQLite: `data/accounting.db`
- Uploaded PDFs: `data/uploads/`
