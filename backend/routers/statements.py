"""Statement API endpoints - file upload and import."""
from fastapi import APIRouter, Depends, UploadFile, File, HTTPException
from sqlalchemy.orm import Session
from sqlalchemy.exc import IntegrityError

from ..database import get_db
from ..config import UPLOADS_DIR
from ..models import Statement, Account, Category
from ..schemas import StatementResponse, ImportResult
from ..services import ImportService
from ..parsers import HSBCPDFParser, StarlingCSVParser

router = APIRouter(prefix="/api/statements", tags=["statements"])


def ensure_account(db: Session, account_id: str, name: str, bank: str) -> Account:
    """Create account if not exists."""
    account = db.query(Account).filter(Account.id == account_id).first()
    if not account:
        account = Account(
            id=account_id,
            name=name,
            bank=bank,
            account_type="current",
            currency="GBP",
        )
        db.add(account)
        db.commit()
    return account


@router.get("", response_model=list[StatementResponse])
def list_statements(db: Session = Depends(get_db)):
    """List all imported statements."""
    statements = db.query(Statement).order_by(Statement.imported_at.desc()).all()

    result = []
    for stmt in statements:
        resp = StatementResponse(
            id=stmt.id,
            account_id=stmt.account_id,
            filename=stmt.filename,
            file_hash=stmt.file_hash,
            period_start=stmt.period_start,
            period_end=stmt.period_end,
            opening_balance=stmt.opening_balance,
            closing_balance=stmt.closing_balance,
            imported_at=stmt.imported_at,
            transaction_count=len(stmt.transactions),
        )
        result.append(resp)

    return result


@router.post("/upload", response_model=ImportResult)
async def upload_statement(
    file: UploadFile = File(...),
    db: Session = Depends(get_db),
):
    """Upload and import a bank statement (PDF or CSV)."""
    if not file.filename:
        raise HTTPException(status_code=400, detail="No file provided")

    filename_lower = file.filename.lower()

    # Determine file type and parser
    if filename_lower.endswith(".pdf"):
        parser = HSBCPDFParser()
        account_id = "hsbc-main"
        account_name = "HSBC Current Account"
        bank = "HSBC"
    elif filename_lower.endswith(".csv"):
        # Detect bank from filename
        if "starling" in filename_lower:
            parser = StarlingCSVParser()
            account_id = "starling-main"
            account_name = "Starling Current Account"
            bank = "Starling"
        else:
            raise HTTPException(
                status_code=400,
                detail="Unknown CSV format. Supported: Starling (filename must contain 'starling')"
            )
    else:
        raise HTTPException(status_code=400, detail="Only PDF and CSV files are supported")

    # Read file content
    content = await file.read()

    # Initialize services
    import_service = ImportService(db)

    # Compute file hash for deduplication
    file_hash = import_service.compute_file_hash(content)

    # Check for duplicate
    existing = db.query(Statement).filter(Statement.file_hash == file_hash).first()
    if existing:
        return ImportResult(
            success=False,
            statement_id=existing.id,
            message=f"This statement was already imported on {existing.imported_at.strftime('%Y-%m-%d')}",
        )

    # Ensure account exists
    ensure_account(db, account_id, account_name, bank)

    # Parse file
    try:
        parsed = parser.parse(content)
    except Exception as e:
        return ImportResult(
            success=False,
            errors=[str(e)],
            message=f"Failed to parse file: {str(e)}",
        )

    # Save file to uploads directory
    safe_name = file.filename.rsplit("/", 1)[-1].rsplit("\\", 1)[-1]
    upload_path = UPLOADS_DIR / f"{file_hash}_{safe_name}"
    with open(upload_path, "wb") as f:
        f.write(content)

    # Create statement record
    statement = Statement(
        account_id=account_id,
        filename=file.filename,
        file_hash=file_hash,
        period_start=parsed["period_start"],
        period_end=parsed["period_end"],
        opening_balance=parsed.get("opening_balance"),
        closing_balance=parsed.get("closing_balance"),
        raw_text=parsed.get("raw_text"),
    )

    try:
        db.add(statement)
        db.commit()
        db.refresh(statement)
    except IntegrityError:
        db.rollback()
        return ImportResult(
            success=False,
            message="Failed to create statement record",
        )

    # For Starling, pre-map categories from their system
    category_map = {}
    if bank == "Starling":
        categories = db.query(Category).all()
        category_map = {c.name: c.id for c in categories}

    # Import transactions
    result = import_service.import_transactions(
        account_id=account_id,
        statement=statement,
        transactions_data=parsed["transactions"],
        category_map=category_map,
    )

    return result


@router.get("/{statement_id}", response_model=StatementResponse)
def get_statement(statement_id: int, db: Session = Depends(get_db)):
    """Get a single statement by ID."""
    stmt = db.query(Statement).filter(Statement.id == statement_id).first()
    if not stmt:
        raise HTTPException(status_code=404, detail="Statement not found")

    return StatementResponse(
        id=stmt.id,
        account_id=stmt.account_id,
        filename=stmt.filename,
        file_hash=stmt.file_hash,
        period_start=stmt.period_start,
        period_end=stmt.period_end,
        opening_balance=stmt.opening_balance,
        closing_balance=stmt.closing_balance,
        imported_at=stmt.imported_at,
        transaction_count=len(stmt.transactions),
    )
