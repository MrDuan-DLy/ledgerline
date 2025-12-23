"""Receipt API endpoints - image upload and OCR."""
import json
from datetime import date
from fastapi import APIRouter, Depends, UploadFile, File, HTTPException
from sqlalchemy.orm import Session
from sqlalchemy.exc import IntegrityError

from ..database import get_db
from ..config import RECEIPTS_DIR
from ..models import Receipt, ReceiptItem
from ..schemas import ReceiptResponse, ReceiptUploadResult, ReceiptConfirmRequest
from ..services.receipt_service import ReceiptService

router = APIRouter(prefix="/api/receipts", tags=["receipts"])


def _to_response(receipt: Receipt) -> ReceiptResponse:
    return ReceiptResponse(
        id=receipt.id,
        image_path=receipt.image_path,
        image_hash=receipt.image_hash,
        merchant_name=receipt.merchant_name,
        receipt_date=receipt.receipt_date,
        receipt_time=receipt.receipt_time,
        total_amount=receipt.total_amount,
        currency=receipt.currency,
        payment_method=receipt.payment_method,
        status=receipt.status,
        ocr_raw=receipt.ocr_raw,
        ocr_json=receipt.ocr_json,
        transaction_id=receipt.transaction_id,
        created_at=receipt.created_at,
        items=[
            {
                "id": item.id,
                "name": item.name,
                "quantity": item.quantity,
                "unit_price": item.unit_price,
                "line_total": item.line_total,
            }
            for item in receipt.items
        ],
    )


@router.get("", response_model=list[ReceiptResponse])
def list_receipts(db: Session = Depends(get_db)):
    """List receipts, newest first."""
    receipts = db.query(Receipt).order_by(Receipt.created_at.desc()).all()
    return [_to_response(r) for r in receipts]


@router.get("/{receipt_id}", response_model=ReceiptResponse)
def get_receipt(receipt_id: int, db: Session = Depends(get_db)):
    receipt = db.query(Receipt).filter(Receipt.id == receipt_id).first()
    if not receipt:
        raise HTTPException(status_code=404, detail="Receipt not found")
    return _to_response(receipt)


@router.post("/upload", response_model=ReceiptUploadResult)
async def upload_receipt(
    file: UploadFile = File(...),
    db: Session = Depends(get_db),
):
    """Upload a receipt image and parse via OCR."""
    if not file.filename:
        raise HTTPException(status_code=400, detail="No file provided")

    content = await file.read()
    mime_type = file.content_type or "image/jpeg"

    service = ReceiptService(db)
    image_hash = service.compute_image_hash(content)

    existing = db.query(Receipt).filter(Receipt.image_hash == image_hash).first()
    if existing:
        return ReceiptUploadResult(
            success=False,
            receipt_id=existing.id,
            message="This receipt was already uploaded",
            errors=[],
        )

    safe_name = file.filename.rsplit("/", 1)[-1].rsplit("\\", 1)[-1]
    image_path = RECEIPTS_DIR / f"{image_hash}_{safe_name}"
    with open(image_path, "wb") as f:
        f.write(content)

    receipt = Receipt(
        image_path=str(image_path),
        image_hash=image_hash,
        status="pending",
    )

    try:
        db.add(receipt)
        db.commit()
        db.refresh(receipt)
    except IntegrityError:
        db.rollback()
        return ReceiptUploadResult(
            success=False,
            message="Failed to create receipt record",
            errors=[],
        )

    return service.parse_receipt(receipt, content, mime_type)


@router.post("/{receipt_id}/confirm")
def confirm_receipt(
    receipt_id: int,
    payload: ReceiptConfirmRequest,
    db: Session = Depends(get_db),
):
    """Confirm receipt and create a transaction."""
    receipt = db.query(Receipt).filter(Receipt.id == receipt_id).first()
    if not receipt:
        raise HTTPException(status_code=404, detail="Receipt not found")

    service = ReceiptService(db)
    overrides = payload.model_dump()
    transaction = service.confirm_receipt(receipt, overrides)
    if not transaction:
        raise HTTPException(status_code=400, detail="Receipt missing required fields")

    return {"transaction_id": transaction.id, "receipt_id": receipt.id}
