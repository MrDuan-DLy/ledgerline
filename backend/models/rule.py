"""Rule model - automatic classification rules."""
from datetime import datetime
from sqlalchemy import String, DateTime, Integer, Boolean, ForeignKey
from sqlalchemy.orm import Mapped, mapped_column, relationship

from ..database import Base


class Rule(Base):
    """Classification rule entity."""

    __tablename__ = "rules"

    id: Mapped[int] = mapped_column(primary_key=True, autoincrement=True)

    # Pattern matching
    pattern: Mapped[str] = mapped_column(String(255), nullable=False)
    pattern_type: Mapped[str] = mapped_column(String(20), default="contains")  # 'contains', 'regex', 'exact'

    # Target category
    category_id: Mapped[int] = mapped_column(ForeignKey("categories.id"), nullable=False)

    # Priority (higher = matched first)
    priority: Mapped[int] = mapped_column(Integer, default=0)
    is_active: Mapped[bool] = mapped_column(Boolean, default=True)

    # Origin tracking (if created from user correction)
    created_from_txn_id: Mapped[int | None] = mapped_column(Integer, nullable=True)

    created_at: Mapped[datetime] = mapped_column(DateTime, default=datetime.utcnow)

    # Relationships
    category = relationship("Category", back_populates="rules")
