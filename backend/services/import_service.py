"""Import service - handles statement parsing and transaction creation."""
import hashlib
from datetime import date
from sqlalchemy.orm import Session
from sqlalchemy.exc import IntegrityError

from ..models import Statement, Transaction, Account
from ..schemas import ImportResult
from .classify_service import ClassifyService


class ImportService:
    """Handles importing bank statements."""

    def __init__(self, db: Session):
        self.db = db
        self.classify_service = ClassifyService(db)

    def compute_file_hash(self, content: bytes) -> str:
        """Compute SHA256 hash of file content."""
        return hashlib.sha256(content).hexdigest()

    def compute_transaction_hash(
        self,
        txn_date: date,
        description: str,
        amount: float,
        balance: float | None
    ) -> str:
        """Compute unique hash for a transaction to detect duplicates."""
        balance_str = f"{balance:.2f}" if balance is not None else "null"
        data = f"{txn_date.isoformat()}|{description}|{amount:.2f}|{balance_str}"
        return hashlib.sha256(data.encode()).hexdigest()

    def ensure_default_account(self) -> Account:
        """Create default HSBC account if not exists."""
        account = self.db.query(Account).filter(Account.id == "hsbc-main").first()
        if not account:
            account = Account(
                id="hsbc-main",
                name="HSBC Current Account",
                bank="HSBC",
                account_type="current",
                currency="GBP",
            )
            self.db.add(account)
            self.db.commit()
        return account

    def import_transactions(
        self,
        account_id: str,
        statement: Statement,
        transactions_data: list[dict],
        category_map: dict[str, int] | None = None,
    ) -> ImportResult:
        """
        Import parsed transactions into database.

        Args:
            account_id: Target account ID
            statement: Statement record (already created)
            transactions_data: List of transaction dicts from parser
            category_map: Optional map of category names to IDs (for pre-mapped categories)

        Returns:
            ImportResult with counts and any errors
        """
        imported = 0
        skipped = 0
        errors = []

        for txn_data in transactions_data:
            try:
                # Compute hash for deduplication
                source_hash = self.compute_transaction_hash(
                    txn_data["date"],
                    txn_data["description"],
                    txn_data["amount"],
                    txn_data.get("balance"),
                )

                # Check for duplicate
                existing = (
                    self.db.query(Transaction)
                    .filter(Transaction.source_hash == source_hash)
                    .first()
                )

                if existing:
                    skipped += 1
                    continue

                # Create transaction
                txn = Transaction(
                    statement_id=statement.id,
                    source_hash=source_hash,
                    raw_date=txn_data["date"],
                    raw_description=txn_data["description"],
                    raw_amount=txn_data["amount"],
                    raw_balance=txn_data.get("balance"),
                    amount=txn_data["amount"],
                    description=txn_data["description"],  # initially same as raw
                    notes=txn_data.get("notes"),
                )

                # Try to use pre-mapped category (e.g., from Starling)
                mapped_cat = txn_data.get("mapped_category")
                if mapped_cat and category_map and mapped_cat in category_map:
                    txn.category_id = category_map[mapped_cat]
                    txn.category_source = "merchant"  # From bank's categorization
                else:
                    # Fall back to rule-based classification
                    self.classify_service.classify_transaction(txn)

                self.db.add(txn)
                imported += 1

            except Exception as e:
                errors.append(f"Row error: {str(e)}")

        try:
            self.db.commit()
        except IntegrityError as e:
            self.db.rollback()
            return ImportResult(
                success=False,
                statement_id=statement.id,
                transactions_imported=0,
                transactions_skipped=0,
                errors=[f"Database error: {str(e)}"],
                message="Import failed due to database error",
            )

        return ImportResult(
            success=True,
            statement_id=statement.id,
            transactions_imported=imported,
            transactions_skipped=skipped,
            errors=errors,
            message=f"Imported {imported} transactions, skipped {skipped} duplicates",
        )
