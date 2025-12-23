"""Statement schemas."""
from datetime import date, datetime
from pydantic import BaseModel


class StatementCreate(BaseModel):
    """Fields for creating a statement record."""
    account_id: str
    filename: str
    file_hash: str
    period_start: date
    period_end: date
    opening_balance: float | None = None
    closing_balance: float | None = None
    raw_text: str | None = None


class StatementResponse(BaseModel):
    """Statement response."""
    id: int
    account_id: str
    filename: str
    file_hash: str
    period_start: date
    period_end: date
    opening_balance: float | None
    closing_balance: float | None
    imported_at: datetime
    transaction_count: int = 0

    class Config:
        from_attributes = True


class ImportResult(BaseModel):
    """Result of importing a statement."""
    success: bool
    statement_id: int | None = None
    transactions_imported: int = 0
    transactions_skipped: int = 0  # duplicates
    errors: list[str] = []
    message: str
