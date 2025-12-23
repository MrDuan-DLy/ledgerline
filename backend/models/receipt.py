"""Receipt model - OCR extracted receipt data."""
from datetime import datetime, date
from sqlalchemy import String, DateTime, Date, Float, Text, ForeignKey
from sqlalchemy.orm import Mapped, mapped_column, relationship

from ..database import Base


class Receipt(Base):
    """Receipt entity for uploaded images and OCR results."""

    __tablename__ = "receipts"

    id: Mapped[int] = mapped_column(primary_key=True, autoincrement=True)
    image_path: Mapped[str] = mapped_column(String(255), nullable=False)
    image_hash: Mapped[str] = mapped_column(String(64), unique=True, nullable=False)

    merchant_name: Mapped[str | None] = mapped_column(String(255), nullable=True)
    receipt_date: Mapped[date | None] = mapped_column(Date, nullable=True)
    receipt_time: Mapped[str | None] = mapped_column(String(20), nullable=True)
    total_amount: Mapped[float | None] = mapped_column(Float, nullable=True)
    currency: Mapped[str | None] = mapped_column(String(10), nullable=True)
    payment_method: Mapped[str | None] = mapped_column(String(50), nullable=True)

    status: Mapped[str] = mapped_column(String(20), default="pending")  # pending, confirmed, failed
    ocr_raw: Mapped[str | None] = mapped_column(Text, nullable=True)
    ocr_json: Mapped[str | None] = mapped_column(Text, nullable=True)

    transaction_id: Mapped[int | None] = mapped_column(ForeignKey("transactions.id"), nullable=True)
    matched_transaction_id: Mapped[int | None] = mapped_column(
        ForeignKey("transactions.id"), nullable=True
    )
    matched_reason: Mapped[str | None] = mapped_column(String(255), nullable=True)

    created_at: Mapped[datetime] = mapped_column(DateTime, default=datetime.utcnow)

    items = relationship("ReceiptItem", back_populates="receipt", cascade="all, delete-orphan")
    transaction = relationship("Transaction", foreign_keys=[transaction_id])
