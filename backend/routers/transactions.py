"""Transaction API endpoints."""
from datetime import date
from fastapi import APIRouter, Depends, Query, HTTPException
from sqlalchemy.orm import Session
from sqlalchemy import func, case

from ..database import get_db
from ..models import Transaction, Category
from ..schemas import (
    TransactionResponse,
    TransactionUpdate,
    TransactionListResponse,
    StatsSeriesResponse,
    DailySeriesPoint,
    CategoryTotal,
)

router = APIRouter(prefix="/api/transactions", tags=["transactions"])


def _to_response(txn: Transaction) -> TransactionResponse:
    """Convert Transaction model to response schema."""
    return TransactionResponse(
        id=txn.id,
        statement_id=txn.statement_id,
        source_hash=txn.source_hash,
        raw_date=txn.raw_date,
        raw_description=txn.raw_description,
        raw_amount=txn.raw_amount,
        raw_balance=txn.raw_balance,
        effective_date=txn.effective_date,
        description=txn.description,
        amount=txn.amount,
        category_id=txn.category_id,
        category_name=txn.category.name if txn.category else None,
        category_source=txn.category_source,
        is_reconciled=txn.is_reconciled,
        reconciled_at=txn.reconciled_at,
        notes=txn.notes,
        created_at=txn.created_at,
        updated_at=txn.updated_at,
    )


@router.get("", response_model=TransactionListResponse)
def list_transactions(
    page: int = Query(1, ge=1),
    page_size: int = Query(50, ge=1, le=200),
    start_date: date | None = None,
    end_date: date | None = None,
    category_id: int | None = None,
    unclassified_only: bool = False,
    search: str | None = None,
    db: Session = Depends(get_db),
):
    """List transactions with filtering and pagination."""
    query = db.query(Transaction)

    # Filters
    if start_date:
        query = query.filter(Transaction.raw_date >= start_date)
    if end_date:
        query = query.filter(Transaction.raw_date <= end_date)
    if category_id:
        query = query.filter(Transaction.category_id == category_id)
    if unclassified_only:
        query = query.filter(Transaction.category_id.is_(None))
    if search:
        query = query.filter(Transaction.raw_description.ilike(f"%{search}%"))

    # Count total
    total = query.count()

    # Paginate
    offset = (page - 1) * page_size
    transactions = (
        query.order_by(Transaction.raw_date.desc())
        .offset(offset)
        .limit(page_size)
        .all()
    )

    return TransactionListResponse(
        items=[_to_response(t) for t in transactions],
        total=total,
        page=page,
        page_size=page_size,
        total_pages=(total + page_size - 1) // page_size,
    )


@router.get("/{transaction_id}", response_model=TransactionResponse)
def get_transaction(transaction_id: int, db: Session = Depends(get_db)):
    """Get a single transaction by ID."""
    txn = db.query(Transaction).filter(Transaction.id == transaction_id).first()
    if not txn:
        raise HTTPException(status_code=404, detail="Transaction not found")
    return _to_response(txn)


@router.patch("/{transaction_id}", response_model=TransactionResponse)
def update_transaction(
    transaction_id: int,
    update: TransactionUpdate,
    db: Session = Depends(get_db),
):
    """Update transaction (classification, notes, etc.)."""
    txn = db.query(Transaction).filter(Transaction.id == transaction_id).first()
    if not txn:
        raise HTTPException(status_code=404, detail="Transaction not found")

    # Apply updates
    if update.effective_date is not None:
        txn.effective_date = update.effective_date
    if update.description is not None:
        txn.description = update.description
    if update.category_id is not None:
        # Verify category exists
        category = db.query(Category).filter(Category.id == update.category_id).first()
        if not category:
            raise HTTPException(status_code=400, detail="Category not found")
        txn.category_id = update.category_id
        txn.category_source = "manual"  # Mark as manually classified
    if update.notes is not None:
        txn.notes = update.notes

    db.commit()
    db.refresh(txn)
    return _to_response(txn)


@router.post("/bulk-classify")
def bulk_classify(
    transaction_ids: list[int],
    category_id: int,
    db: Session = Depends(get_db),
):
    """Bulk classify multiple transactions."""
    # Verify category
    category = db.query(Category).filter(Category.id == category_id).first()
    if not category:
        raise HTTPException(status_code=400, detail="Category not found")

    updated = (
        db.query(Transaction)
        .filter(Transaction.id.in_(transaction_ids))
        .update(
            {"category_id": category_id, "category_source": "manual"},
            synchronize_session=False,
        )
    )

    db.commit()
    return {"updated": updated}


@router.get("/stats/summary")
def get_summary(
    start_date: date | None = None,
    end_date: date | None = None,
    db: Session = Depends(get_db),
):
    """Get summary statistics for transactions."""
    query = db.query(Transaction)

    if start_date:
        query = query.filter(Transaction.raw_date >= start_date)
    if end_date:
        query = query.filter(Transaction.raw_date <= end_date)

    total_count = query.count()
    income = query.filter(Transaction.amount > 0).with_entities(func.sum(Transaction.amount)).scalar() or 0
    expenses = query.filter(Transaction.amount < 0).with_entities(func.sum(Transaction.amount)).scalar() or 0
    unclassified = query.filter(Transaction.category_id.is_(None)).count()

    return {
        "total_transactions": total_count,
        "total_income": round(income, 2),
        "total_expenses": round(abs(expenses), 2),
        "net": round(income + expenses, 2),
        "unclassified_count": unclassified,
    }


@router.get("/stats/series", response_model=StatsSeriesResponse)
def get_series(
    start_date: date | None = None,
    end_date: date | None = None,
    db: Session = Depends(get_db),
):
    """Get daily totals and category breakdown for charts."""
    base_query = db.query(Transaction)

    if start_date:
        base_query = base_query.filter(Transaction.raw_date >= start_date)
    if end_date:
        base_query = base_query.filter(Transaction.raw_date <= end_date)

    date_filter = base_query.subquery()

    daily_rows = (
        db.query(
            date_filter.c.raw_date.label("date"),
            func.sum(date_filter.c.amount).label("net"),
            func.sum(
                case((date_filter.c.amount > 0, date_filter.c.amount), else_=0.0)
            ).label("income"),
            func.sum(
                case((date_filter.c.amount < 0, -date_filter.c.amount), else_=0.0)
            ).label("expenses"),
            func.count(date_filter.c.id).label("count"),
        )
        .group_by(date_filter.c.raw_date)
        .order_by(date_filter.c.raw_date)
        .all()
    )

    daily: list[DailySeriesPoint] = []
    cumulative = 0.0
    for row in daily_rows:
        net = float(row.net or 0.0)
        cumulative += net
        daily.append(
            DailySeriesPoint(
                date=row.date,
                net=round(net, 2),
                income=round(float(row.income or 0.0), 2),
                expenses=round(float(row.expenses or 0.0), 2),
                count=row.count,
                cumulative=round(cumulative, 2),
            )
        )

    category_rows = (
        db.query(
            Category.id.label("category_id"),
            Category.name.label("category_name"),
            func.sum(
                case((Transaction.amount < 0, -Transaction.amount), else_=0.0)
            ).label("expenses"),
            func.sum(
                case((Transaction.amount > 0, Transaction.amount), else_=0.0)
            ).label("income"),
            func.sum(Transaction.amount).label("net"),
            func.count(Transaction.id).label("count"),
        )
        .outerjoin(Category, Category.id == Transaction.category_id)
        .filter(Transaction.id.in_(db.query(date_filter.c.id)))
        .group_by(Category.id, Category.name)
        .order_by(func.sum(Transaction.amount).asc())
        .all()
    )

    categories: list[CategoryTotal] = []
    for row in category_rows:
        name = row.category_name if row.category_name else "Unclassified"
        categories.append(
            CategoryTotal(
                category_id=row.category_id,
                category_name=name,
                expenses=round(float(row.expenses or 0.0), 2),
                income=round(float(row.income or 0.0), 2),
                net=round(float(row.net or 0.0), 2),
                count=row.count,
            )
        )

    return StatsSeriesResponse(
        start_date=start_date,
        end_date=end_date,
        daily=daily,
        categories=categories,
    )
