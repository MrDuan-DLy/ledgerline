-- +goose Up

CREATE TABLE IF NOT EXISTS accounts (
    id          TEXT PRIMARY KEY,
    name        TEXT    NOT NULL,
    bank        TEXT    NOT NULL,
    account_type TEXT   NOT NULL,
    currency    TEXT    DEFAULT 'GBP',
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS statements (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id      TEXT    NOT NULL REFERENCES accounts(id),
    filename        TEXT    NOT NULL,
    file_hash       TEXT    UNIQUE NOT NULL,
    period_start    DATE    NOT NULL,
    period_end      DATE    NOT NULL,
    opening_balance REAL,
    closing_balance REAL,
    raw_text        TEXT,
    imported_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS categories (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    UNIQUE NOT NULL,
    parent_id   INTEGER REFERENCES categories(id),
    icon        TEXT,
    color       TEXT,
    is_expense  BOOLEAN DEFAULT 1
);

CREATE TABLE IF NOT EXISTS transactions (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    statement_id    INTEGER REFERENCES statements(id),
    source_hash     TEXT    UNIQUE NOT NULL,
    raw_date        DATE    NOT NULL,
    raw_description TEXT    NOT NULL,
    raw_amount      REAL    NOT NULL,
    raw_balance     REAL,
    effective_date  DATE,
    description     TEXT,
    amount          REAL    NOT NULL,
    category_id     INTEGER REFERENCES categories(id),
    category_source TEXT    DEFAULT 'unclassified',
    is_excluded     BOOLEAN DEFAULT 0,
    is_reconciled   BOOLEAN DEFAULT 0,
    reconciled_at   DATETIME,
    notes           TEXT,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS rules (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    pattern             TEXT    NOT NULL,
    pattern_type        TEXT    DEFAULT 'contains',
    category_id         INTEGER NOT NULL REFERENCES categories(id),
    priority            INTEGER DEFAULT 0,
    is_active           BOOLEAN DEFAULT 1,
    created_from_txn_id INTEGER,
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS audit_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    table_name  TEXT    NOT NULL,
    record_id   INTEGER NOT NULL,
    action      TEXT    NOT NULL,
    old_values  TEXT,
    new_values  TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS receipts (
    id                      INTEGER PRIMARY KEY AUTOINCREMENT,
    image_path              TEXT    NOT NULL,
    image_hash              TEXT    UNIQUE NOT NULL,
    merchant_name           TEXT,
    receipt_date            DATE,
    receipt_time            TEXT,
    total_amount            REAL,
    currency                TEXT,
    payment_method          TEXT,
    status                  TEXT    DEFAULT 'pending',
    ocr_raw                 TEXT,
    ocr_json                TEXT,
    transaction_id          INTEGER REFERENCES transactions(id),
    matched_transaction_id  INTEGER REFERENCES transactions(id),
    matched_reason          TEXT,
    created_at              DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS receipt_items (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    receipt_id  INTEGER NOT NULL REFERENCES receipts(id),
    name        TEXT    NOT NULL,
    quantity    REAL,
    unit_price  REAL,
    line_total  REAL
);

CREATE TABLE IF NOT EXISTS merchants (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    UNIQUE NOT NULL,
    patterns    TEXT    DEFAULT '[]',
    category_id INTEGER REFERENCES categories(id),
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS budgets (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    category_id     INTEGER UNIQUE NOT NULL REFERENCES categories(id),
    monthly_limit   REAL    NOT NULL,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS import_sessions (
    id              TEXT PRIMARY KEY,
    source_type     TEXT    NOT NULL,
    source_file     TEXT    NOT NULL,
    file_hash       TEXT    NOT NULL,
    file_path       TEXT    NOT NULL,
    page_count      INTEGER,
    page_image_paths TEXT,
    metadata_json   TEXT,
    ai_usage_json   TEXT,
    status          TEXT    DEFAULT 'pending',
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS import_items (
    id                      INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id              TEXT    NOT NULL REFERENCES import_sessions(id),
    page_num                INTEGER,
    extracted_date          DATE,
    extracted_description   TEXT,
    extracted_amount        REAL,
    extracted_balance       REAL,
    extracted_merchant      TEXT,
    extracted_items_json    TEXT,
    raw_ai_json             TEXT,
    status                  TEXT    DEFAULT 'pending',
    duplicate_of_id         INTEGER REFERENCES transactions(id),
    duplicate_score         REAL,
    duplicate_reason        TEXT,
    created_at              DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS import_items;
DROP TABLE IF EXISTS import_sessions;
DROP TABLE IF EXISTS budgets;
DROP TABLE IF EXISTS merchants;
DROP TABLE IF EXISTS receipt_items;
DROP TABLE IF EXISTS receipts;
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS rules;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS statements;
DROP TABLE IF EXISTS accounts;
