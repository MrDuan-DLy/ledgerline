"""Account model - represents a bank account."""
from datetime import datetime
from sqlalchemy import String, DateTime
from sqlalchemy.orm import Mapped, mapped_column, relationship

from ..database import Base


class Account(Base):
    """Bank account entity."""

    __tablename__ = "accounts"

    id: Mapped[str] = mapped_column(String(50), primary_key=True)  # e.g., 'hsbc-main'
    name: Mapped[str] = mapped_column(String(100), nullable=False)
    bank: Mapped[str] = mapped_column(String(50), nullable=False)  # e.g., 'HSBC'
    account_type: Mapped[str] = mapped_column(String(20), nullable=False)  # 'current', 'savings', 'credit'
    currency: Mapped[str] = mapped_column(String(3), default="GBP")
    created_at: Mapped[datetime] = mapped_column(DateTime, default=datetime.utcnow)

    # Relationships
    statements = relationship("Statement", back_populates="account")
