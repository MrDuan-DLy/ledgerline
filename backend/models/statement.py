"""Statement model - represents an imported bank statement file."""
from datetime import datetime, date
from sqlalchemy import String, DateTime, Date, Float, Text, ForeignKey
from sqlalchemy.orm import Mapped, mapped_column, relationship

from ..database import Base


class Statement(Base):
    """Bank statement file entity - tracks imported PDFs/CSVs."""

    __tablename__ = "statements"

    id: Mapped[int] = mapped_column(primary_key=True, autoincrement=True)
    account_id: Mapped[str] = mapped_column(String(50), ForeignKey("accounts.id"), nullable=False)

    # File tracking
    filename: Mapped[str] = mapped_column(String(255), nullable=False)
    file_hash: Mapped[str] = mapped_column(String(64), unique=True, nullable=False)  # SHA256

    # Statement period
    period_start: Mapped[date] = mapped_column(Date, nullable=False)
    period_end: Mapped[date] = mapped_column(Date, nullable=False)

    # Balance verification
    opening_balance: Mapped[float | None] = mapped_column(Float, nullable=True)
    closing_balance: Mapped[float | None] = mapped_column(Float, nullable=True)

    # Raw data preservation
    raw_text: Mapped[str | None] = mapped_column(Text, nullable=True)

    imported_at: Mapped[datetime] = mapped_column(DateTime, default=datetime.utcnow)

    # Relationships
    account = relationship("Account", back_populates="statements")
    transactions = relationship("Transaction", back_populates="statement")
