"""Database connection and session management."""
from sqlalchemy import create_engine, event, text
from sqlalchemy.orm import sessionmaker, DeclarativeBase

from .config import DATABASE_URL


class Base(DeclarativeBase):
    """Base class for all ORM models."""
    pass


engine = create_engine(
    DATABASE_URL,
    connect_args={"check_same_thread": False},  # SQLite specific
    echo=False,
)

# Enable foreign keys for SQLite
@event.listens_for(engine, "connect")
def set_sqlite_pragma(dbapi_connection, connection_record):
    cursor = dbapi_connection.cursor()
    cursor.execute("PRAGMA foreign_keys=ON")
    cursor.close()


SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)


def get_db():
    """Dependency for FastAPI to get database session."""
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()


def init_db():
    """Create all tables."""
    Base.metadata.create_all(bind=engine)
    _ensure_receipt_columns()


def _ensure_receipt_columns():
    """Add new receipt columns when using an existing SQLite database."""
    with engine.connect() as conn:
        table = conn.execute(
            text("SELECT name FROM sqlite_master WHERE type='table' AND name='receipts'")
        ).fetchone()
        if not table:
            return

        columns = conn.execute(text("PRAGMA table_info(receipts)")).fetchall()
        existing = {row[1] for row in columns}

        if "matched_transaction_id" not in existing:
            conn.execute(
                text("ALTER TABLE receipts ADD COLUMN matched_transaction_id INTEGER")
            )
        if "matched_reason" not in existing:
            conn.execute(
                text("ALTER TABLE receipts ADD COLUMN matched_reason VARCHAR(255)")
            )
