"""Receipt API endpoints - image upload and OCR."""
from fastapi import APIRouter, Depends, UploadFile, File, HTTPException
from sqlalchemy.orm import Session
from sqlalchemy.exc import IntegrityError

from ..database import get_db
from ..config import RECEIPTS_DIR
from ..models import Receipt, Transaction
from ..schemas import ReceiptResponse, ReceiptUploadResult, ReceiptConfirmRequest
from ..services.receipt_service import ReceiptService

router = APIRouter(prefix="/api/receipts", tags=["receipts"])


def _to_response(receipt: Receipt, matched_txn: Transaction | None = None) -> ReceiptResponse:
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
        matched_transaction_id=receipt.matched_transaction_id,
        matched_transaction_date=matched_txn.raw_date if matched_txn else None,
        matched_transaction_amount=matched_txn.amount if matched_txn else None,
        matched_transaction_description=matched_txn.raw_description if matched_txn else None,
        matched_reason=receipt.matched_reason,
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
    match_ids = [r.matched_transaction_id for r in receipts if r.matched_transaction_id]
    matched = {}
    if match_ids:
        matched_rows = db.query(Transaction).filter(Transaction.id.in_(match_ids)).all()
        matched = {t.id: t for t in matched_rows}
    return [_to_response(r, matched.get(r.matched_transaction_id)) for r in receipts]


@router.get("/{receipt_id}", response_model=ReceiptResponse)
def get_receipt(receipt_id: int, db: Session = Depends(get_db)):
    receipt = db.query(Receipt).filter(Receipt.id == receipt_id).first()
    if not receipt:
        raise HTTPException(status_code=404, detail="Receipt not found")
    matched_txn = None
    if receipt.matched_transaction_id:
        matched_txn = (
            db.query(Transaction)
            .filter(Transaction.id == receipt.matched_transaction_id)
            .first()
        )
    return _to_response(receipt, matched_txn)


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


@router.post("/upload-batch", response_model=list[ReceiptUploadResult])
async def upload_receipts_batch(
    files: list[UploadFile] = File(...),
    db: Session = Depends(get_db),
):
    """Upload multiple receipt images and parse via OCR."""
    service = ReceiptService(db)
    results: list[ReceiptUploadResult] = []

    for file in files:
        if not file.filename:
            results.append(
                ReceiptUploadResult(
                    success=False,
                    receipt_id=None,
                    message="No file provided",
                    errors=[],
                )
            )
            continue

        content = await file.read()
        mime_type = file.content_type or "image/jpeg"
        image_hash = service.compute_image_hash(content)

        existing = db.query(Receipt).filter(Receipt.image_hash == image_hash).first()
        if existing:
            results.append(
                ReceiptUploadResult(
                    success=False,
                    receipt_id=existing.id,
                    message="This receipt was already uploaded",
                    errors=[],
                )
            )
            continue

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
            results.append(
                ReceiptUploadResult(
                    success=False,
                    receipt_id=None,
                    message="Failed to create receipt record",
                    errors=[],
                )
            )
            continue

        results.append(service.parse_receipt(receipt, content, mime_type))

    return results


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
        if payload.transaction_id:
            raise HTTPException(status_code=400, detail="Matched transaction not found")
        raise HTTPException(status_code=400, detail="Receipt missing required fields")

    return {"transaction_id": transaction.id, "receipt_id": receipt.id}
