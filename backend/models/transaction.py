"""Transaction model - core entity for all financial transactions."""
from datetime import datetime, date
from sqlalchemy import String, DateTime, Date, Float, Text, Boolean, ForeignKey
from sqlalchemy.orm import Mapped, mapped_column, relationship

from ..database import Base


class Transaction(Base):
    """Financial transaction entity."""

    __tablename__ = "transactions"

    id: Mapped[int] = mapped_column(primary_key=True, autoincrement=True)

    # Source traceability (NEVER auto-overwrite)
    statement_id: Mapped[int | None] = mapped_column(ForeignKey("statements.id"), nullable=True)
    source_hash: Mapped[str] = mapped_column(String(64), unique=True, nullable=False)

    # Raw data from statement (immutable after import)
    raw_date: Mapped[date] = mapped_column(Date, nullable=False)
    raw_description: Mapped[str] = mapped_column(Text, nullable=False)
    raw_amount: Mapped[float] = mapped_column(Float, nullable=False)  # negative=expense, positive=income
    raw_balance: Mapped[float | None] = mapped_column(Float, nullable=True)

    # Derived/correctable data
    effective_date: Mapped[date | None] = mapped_column(Date, nullable=True)  # user can override
    description: Mapped[str | None] = mapped_column(Text, nullable=True)  # cleaned description
    amount: Mapped[float] = mapped_column(Float, nullable=False)

    # Classification
    category_id: Mapped[int | None] = mapped_column(ForeignKey("categories.id"), nullable=True)
    category_source: Mapped[str] = mapped_column(
        String(20), default="unclassified"
    )  # 'manual', 'rule', 'merchant', 'unclassified'

    # Reconciliation
    is_reconciled: Mapped[bool] = mapped_column(Boolean, default=False)
    reconciled_at: Mapped[datetime | None] = mapped_column(DateTime, nullable=True)

    # Metadata
    notes: Mapped[str | None] = mapped_column(Text, nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=datetime.utcnow)
    updated_at: Mapped[datetime] = mapped_column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # Relationships
    statement = relationship("Statement", back_populates="transactions")
    category = relationship("Category", back_populates="transactions")
