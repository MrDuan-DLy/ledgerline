"""Receipt schemas."""
from datetime import date, datetime
from pydantic import BaseModel


class ReceiptItemResponse(BaseModel):
    """Receipt line item response."""
    id: int
    name: str
    quantity: float | None = None
    unit_price: float | None = None
    line_total: float | None = None

    class Config:
        from_attributes = True


class ReceiptResponse(BaseModel):
    """Receipt response."""
    id: int
    image_path: str
    image_hash: str
    merchant_name: str | None
    receipt_date: date | None
    receipt_time: str | None
    total_amount: float | None
    currency: str | None
    payment_method: str | None
    status: str
    ocr_raw: str | None
    ocr_json: str | None
    transaction_id: int | None
    created_at: datetime
    items: list[ReceiptItemResponse] = []

    class Config:
        from_attributes = True


class ReceiptUploadResult(BaseModel):
    """Result of uploading and parsing a receipt."""
    success: bool
    receipt_id: int | None = None
    message: str
    errors: list[str] = []


class ReceiptConfirmRequest(BaseModel):
    """Confirm receipt into a transaction."""
    merchant_name: str | None = None
    receipt_date: date | None = None
    total_amount: float | None = None
    currency: str | None = None
    category_id: int | None = None
    notes: str | None = None
