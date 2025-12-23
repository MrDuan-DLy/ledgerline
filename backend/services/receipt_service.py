"""Receipt OCR service using Gemini."""
import base64
import json
import os
import hashlib
import urllib.request
from datetime import date, datetime

from ..models import Receipt, ReceiptItem, Transaction
from ..schemas import ReceiptUploadResult
from .import_service import ImportService


class ReceiptService:
    """Handles receipt OCR and confirmation."""

    def __init__(self, db):
        self.db = db
        self.import_service = ImportService(db)

    def compute_image_hash(self, content: bytes) -> str:
        """Compute SHA256 hash of image content."""
        return hashlib.sha256(content).hexdigest()

    def _gemini_request(self, content: bytes, mime_type: str) -> tuple[bool, str | None]:
        """Call Gemini API and return raw text response."""
        api_key = os.getenv("GEMINI_API_KEY")
        if not api_key:
            return False, "Missing GEMINI_API_KEY"

        model = os.getenv("GEMINI_MODEL", "gemini-1.5-flash")
        url = (
            "https://generativelanguage.googleapis.com/v1beta/models/"
            f"{model}:generateContent?key={api_key}"
        )

        prompt = (
            "Extract receipt data and output ONLY valid JSON with keys: "
            "merchant_name, receipt_date (YYYY-MM-DD), receipt_time (HH:MM or null), "
            "total_amount (number), currency (e.g. GBP), payment_method, items "
            "(array of {name, quantity, unit_price, line_total}), raw_text."
        )

        payload = {
            "contents": [
                {
                    "role": "user",
                    "parts": [
                        {"text": prompt},
                        {
                            "inlineData": {
                                "mimeType": mime_type,
                                "data": base64.b64encode(content).decode("ascii"),
                            }
                        },
                    ],
                }
            ]
        }

        request = urllib.request.Request(
            url,
            data=json.dumps(payload).encode("utf-8"),
            headers={"Content-Type": "application/json"},
            method="POST",
        )

        try:
            with urllib.request.urlopen(request, timeout=60) as response:
                body = response.read().decode("utf-8")
        except Exception as exc:
            return False, str(exc)

        return True, body

    def _extract_json(self, text: str) -> tuple[dict | None, str | None]:
        """Extract JSON object from Gemini text response."""
        try:
            payload = json.loads(text)
        except json.JSONDecodeError:
            payload = None

        if isinstance(payload, dict):
            candidates = payload.get("candidates", [])
            if not candidates:
                return None, "No candidates in response"
            parts = candidates[0].get("content", {}).get("parts", [])
            if not parts:
                return None, "No content parts in response"
            content_text = parts[0].get("text", "")
        else:
            content_text = text

        start = content_text.find("{")
        end = content_text.rfind("}")
        if start == -1 or end == -1 or start >= end:
            return None, "Failed to locate JSON in response"

        try:
            return json.loads(content_text[start : end + 1]), None
        except json.JSONDecodeError as exc:
            return None, str(exc)

    def _parse_date(self, value: str | None) -> date | None:
        if not value:
            return None
        try:
            return datetime.strptime(value, "%Y-%m-%d").date()
        except ValueError:
            return None

    def parse_receipt(self, receipt: Receipt, content: bytes, mime_type: str) -> ReceiptUploadResult:
        """Parse receipt image via Gemini and update receipt record."""
        ok, response = self._gemini_request(content, mime_type)
        if not ok or response is None:
            receipt.status = "failed"
            self.db.commit()
            return ReceiptUploadResult(
                success=False,
                receipt_id=receipt.id,
                message="OCR request failed",
                errors=[response or "Unknown error"],
            )

        data, error = self._extract_json(response)
        if error or not data:
            receipt.status = "failed"
            receipt.ocr_raw = response
            self.db.commit()
            return ReceiptUploadResult(
                success=False,
                receipt_id=receipt.id,
                message="Failed to parse OCR response",
                errors=[error or "Unknown parse error"],
            )

        receipt.ocr_raw = data.get("raw_text")
        receipt.ocr_json = json.dumps(data)
        receipt.merchant_name = data.get("merchant_name")
        receipt.receipt_date = self._parse_date(data.get("receipt_date"))
        receipt.receipt_time = data.get("receipt_time")
        receipt.total_amount = data.get("total_amount")
        receipt.currency = data.get("currency")
        receipt.payment_method = data.get("payment_method")
        receipt.status = "pending"

        self.db.query(ReceiptItem).filter(ReceiptItem.receipt_id == receipt.id).delete()
        for item in data.get("items", []) or []:
            name = (item.get("name") or "").strip()
            if not name:
                continue
            receipt_item = ReceiptItem(
                receipt_id=receipt.id,
                name=name,
                quantity=item.get("quantity"),
                unit_price=item.get("unit_price"),
                line_total=item.get("line_total"),
            )
            self.db.add(receipt_item)

        self.db.commit()
        return ReceiptUploadResult(
            success=True,
            receipt_id=receipt.id,
            message="Receipt parsed",
            errors=[],
        )

    def confirm_receipt(self, receipt: Receipt, overrides: dict) -> Transaction | None:
        """Create a transaction from a receipt."""
        if receipt.status == "confirmed":
            return receipt.transaction

        merchant_name = overrides.get("merchant_name") or receipt.merchant_name
        receipt_date = overrides.get("receipt_date") or receipt.receipt_date or date.today()
        total_amount = overrides.get("total_amount") or receipt.total_amount
        currency = overrides.get("currency") or receipt.currency or "GBP"
        category_id = overrides.get("category_id")
        notes = overrides.get("notes")

        if total_amount is None or merchant_name is None:
            return None

        amount = -abs(float(total_amount))
        source_hash = self.import_service.compute_transaction_hash(
            receipt_date, merchant_name, amount, None
        )

        existing = (
            self.db.query(Transaction)
            .filter(Transaction.source_hash == source_hash)
            .first()
        )
        if existing:
            receipt.status = "confirmed"
            receipt.transaction_id = existing.id
            self.db.commit()
            return existing

        txn = Transaction(
            statement_id=None,
            source_hash=source_hash,
            raw_date=receipt_date,
            raw_description=merchant_name,
            raw_amount=amount,
            raw_balance=None,
            amount=amount,
            description=merchant_name,
            notes=notes,
            category_id=category_id,
            category_source="receipt" if category_id else "unclassified",
        )
        self.db.add(txn)
        self.db.commit()
        self.db.refresh(txn)

        receipt.status = "confirmed"
        receipt.transaction_id = txn.id
        receipt.ocr_raw = (receipt.ocr_raw or "") + f"\nCurrency: {currency}"
        self.db.commit()

        return txn
