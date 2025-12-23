"""Transaction schemas."""
from datetime import date, datetime
from pydantic import BaseModel


class TransactionBase(BaseModel):
    """Base transaction fields."""
    raw_date: date
    raw_description: str
    raw_amount: float
    raw_balance: float | None = None


class TransactionCreate(TransactionBase):
    """Fields for creating a transaction."""
    statement_id: int | None = None
    source_hash: str


class TransactionUpdate(BaseModel):
    """Fields that can be updated by user."""
    effective_date: date | None = None
    description: str | None = None
    category_id: int | None = None
    notes: str | None = None


class TransactionResponse(BaseModel):
    """Transaction response with all fields."""
    id: int
    statement_id: int | None
    source_hash: str

    # Raw data
    raw_date: date
    raw_description: str
    raw_amount: float
    raw_balance: float | None

    # Derived data
    effective_date: date | None
    description: str | None
    amount: float

    # Classification
    category_id: int | None
    category_name: str | None = None
    category_source: str

    # Status
    is_reconciled: bool
    reconciled_at: datetime | None

    notes: str | None
    created_at: datetime
    updated_at: datetime

    class Config:
        from_attributes = True


class TransactionListResponse(BaseModel):
    """Paginated transaction list."""
    items: list[TransactionResponse]
    total: int
    page: int
    page_size: int
    total_pages: int


class DailySeriesPoint(BaseModel):
    """Daily rollup for charting."""
    date: date
    net: float
    income: float
    expenses: float
    count: int
    cumulative: float


class CategoryTotal(BaseModel):
    """Category totals for charting."""
    category_id: int | None
    category_name: str
    expenses: float
    income: float
    net: float
    count: int


class StatsSeriesResponse(BaseModel):
    """Time series and category breakdown for dashboard."""
    start_date: date | None
    end_date: date | None
    daily: list[DailySeriesPoint]
    categories: list[CategoryTotal]
