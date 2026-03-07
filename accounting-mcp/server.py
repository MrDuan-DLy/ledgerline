"""
Personal Accounting MCP Server
===============================
Read-only query tools over a personal expense tracking SQLite database.

Tools available:
  - list_categories: Show all expense/income categories
  - get_account_overview: Account info and statement coverage
  - get_summary: Spending summary with date filters
  - get_category_breakdown: Top spending categories
  - get_monthly_trend: Monthly spending over time
  - get_budget_status: Budget vs actual spending
  - search_transactions: Find transactions by text/date/category
  - get_merchant_spend: Spending at a specific merchant

Database: SQLite file (accounting.db) in the same directory as this script.

Amount conventions:
  - negative = expense (money out)
  - positive = income/refund (money in)

Category source values:
  - 'manual': user-assigned, highest priority
  - 'rule': assigned by classification rule
  - 'merchant': assigned via merchant mapping
  - 'unclassified': not yet categorized

is_excluded: True for inter-account transfers and other transactions
             that should not count toward spending/income totals.
"""

import json
from datetime import date, datetime
from pathlib import Path

from mcp.server.fastmcp import FastMCP
from sqlalchemy import (
    Boolean,
    Column,
    Date,
    DateTime,
    Float,
    ForeignKey,
    Integer,
    String,
    Text,
    create_engine,
    func,
)
from sqlalchemy.orm import DeclarativeBase, Session, sessionmaker

# ---------------------------------------------------------------------------
# FastMCP instance
# ---------------------------------------------------------------------------
mcp = FastMCP("Personal Accounting")

# ---------------------------------------------------------------------------
# Database setup — SQLite in the same directory as this script
# ---------------------------------------------------------------------------
DB_PATH = Path(__file__).parent / "accounting.db"
engine = create_engine(
    f"sqlite:///{DB_PATH}",
    connect_args={"check_same_thread": False},
    echo=False,
)
SessionLocal = sessionmaker(bind=engine, autocommit=False, autoflush=False)


# ---------------------------------------------------------------------------
# Read-only ORM models (inline, no dependency on backend/)
# Only columns needed for queries are defined here.
# ---------------------------------------------------------------------------
class Base(DeclarativeBase):
    pass


class Account(Base):
    __tablename__ = "accounts"
    id = Column(String(50), primary_key=True)  # e.g. 'hsbc-main'
    name = Column(String(100), nullable=False)
    bank = Column(String(50), nullable=False)
    account_type = Column(String(20), nullable=False)  # current, savings, credit
    currency = Column(String(3), default="GBP")


class Statement(Base):
    __tablename__ = "statements"
    id = Column(Integer, primary_key=True)
    account_id = Column(String(50), ForeignKey("accounts.id"), nullable=False)
    filename = Column(String(255), nullable=False)
    period_start = Column(Date, nullable=False)
    period_end = Column(Date, nullable=False)
    opening_balance = Column(Float, nullable=True)
    closing_balance = Column(Float, nullable=True)
    imported_at = Column(DateTime)


class Category(Base):
    __tablename__ = "categories"
    id = Column(Integer, primary_key=True)
    name = Column(String(50), unique=True, nullable=False)
    parent_id = Column(Integer, ForeignKey("categories.id"), nullable=True)
    is_expense = Column(Boolean, default=True)


class Transaction(Base):
    __tablename__ = "transactions"
    id = Column(Integer, primary_key=True)
    statement_id = Column(Integer, ForeignKey("statements.id"), nullable=True)
    raw_date = Column(Date, nullable=False)
    raw_description = Column(Text, nullable=False)
    raw_amount = Column(Float, nullable=False)
    raw_balance = Column(Float, nullable=True)
    effective_date = Column(Date, nullable=True)
    description = Column(Text, nullable=True)
    amount = Column(Float, nullable=False)
    category_id = Column(Integer, ForeignKey("categories.id"), nullable=True)
    category_source = Column(String(20), default="unclassified")
    is_excluded = Column(Boolean, default=False)
    notes = Column(Text, nullable=True)
    created_at = Column(DateTime)


class Merchant(Base):
    __tablename__ = "merchants"
    id = Column(Integer, primary_key=True)
    name = Column(String(255), unique=True, nullable=False)
    patterns = Column(Text, default="[]")  # JSON list of normalized aliases
    category_id = Column(Integer, ForeignKey("categories.id"), nullable=True)


class Budget(Base):
    __tablename__ = "budgets"
    id = Column(Integer, primary_key=True)
    category_id = Column(Integer, ForeignKey("categories.id"), unique=True, nullable=False)
    monthly_limit = Column(Float, nullable=False)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

# Transfer category names — transactions in these categories are excluded
# from spending/income summaries to avoid double-counting.
TRANSFER_CATEGORY_NAMES = {"Transfer In", "Transfer Out"}


def _get_session() -> Session:
    """Create a new read-only database session."""
    return SessionLocal()


def _get_transfer_category_ids(db: Session) -> list[int]:
    """Return category IDs for transfer categories."""
    rows = db.query(Category.id).filter(Category.name.in_(TRANSFER_CATEGORY_NAMES)).all()
    return [r[0] for r in rows]


def _apply_date_filters(query, start_date: str | None, end_date: str | None):
    """Apply optional date range filters to a transaction query.

    Dates should be ISO format strings (YYYY-MM-DD).
    Uses effective_date if set, otherwise raw_date.
    """
    if start_date:
        d = date.fromisoformat(start_date)
        query = query.filter(
            func.coalesce(Transaction.effective_date, Transaction.raw_date) >= d
        )
    if end_date:
        d = date.fromisoformat(end_date)
        query = query.filter(
            func.coalesce(Transaction.effective_date, Transaction.raw_date) <= d
        )
    return query


def _base_expense_query(db: Session):
    """Return a query on Transaction excluding transfers and excluded rows.

    This is the standard starting point for spending/income analysis.
    """
    transfer_ids = _get_transfer_category_ids(db)
    q = db.query(Transaction).filter(Transaction.is_excluded.is_(False))
    if transfer_ids:
        q = q.filter(~Transaction.category_id.in_(transfer_ids))
    return q


def _serialize(obj) -> str:
    """Serialize a Python object to a JSON string, handling dates."""
    return json.dumps(obj, default=str, ensure_ascii=False)


# ---------------------------------------------------------------------------
# MCP Tools
# ---------------------------------------------------------------------------


@mcp.tool()
def list_categories() -> str:
    """List all transaction categories.

    Returns every category with its id, name, and whether it is an expense
    category. Use this to discover available category names for filtering
    in other tools (e.g. search_transactions, get_monthly_trend).

    Returns:
        JSON array of {id, name, parent_id, is_expense} objects.
    """
    db = _get_session()
    try:
        rows = db.query(Category).order_by(Category.name).all()
        result = [
            {
                "id": c.id,
                "name": c.name,
                "parent_id": c.parent_id,
                "is_expense": c.is_expense,
            }
            for c in rows
        ]
        return _serialize(result)
    finally:
        db.close()


@mcp.tool()
def get_account_overview() -> str:
    """Get an overview of all bank accounts and their data coverage.

    Shows each account with its bank, type, currency, number of imported
    statements, latest closing balance, transaction date range, and total
    transaction count.

    Use this to understand what data is available before running queries.

    Returns:
        JSON array of account overview objects.
    """
    db = _get_session()
    try:
        accounts = db.query(Account).all()
        result = []
        for acct in accounts:
            # Statement stats
            stmts = (
                db.query(Statement)
                .filter(Statement.account_id == acct.id)
                .order_by(Statement.period_end.desc())
                .all()
            )
            # Transaction stats for this account's statements
            stmt_ids = [s.id for s in stmts]
            txn_count = 0
            date_min = None
            date_max = None
            if stmt_ids:
                txn_stats = (
                    db.query(
                        func.count(Transaction.id),
                        func.min(Transaction.raw_date),
                        func.max(Transaction.raw_date),
                    )
                    .filter(Transaction.statement_id.in_(stmt_ids))
                    .first()
                )
                txn_count = txn_stats[0] or 0
                date_min = txn_stats[1]
                date_max = txn_stats[2]

            result.append(
                {
                    "account_id": acct.id,
                    "name": acct.name,
                    "bank": acct.bank,
                    "account_type": acct.account_type,
                    "currency": acct.currency,
                    "statement_count": len(stmts),
                    "latest_balance": stmts[0].closing_balance if stmts else None,
                    "earliest_date": str(date_min) if date_min else None,
                    "latest_date": str(date_max) if date_max else None,
                    "transaction_count": txn_count,
                }
            )
        return _serialize(result)
    finally:
        db.close()


@mcp.tool()
def get_summary(start_date: str | None = None, end_date: str | None = None) -> str:
    """Get a spending summary for a date range.

    Calculates total expenses, total income, net amount, transaction count,
    and how many transactions are still unclassified. Excludes inter-account
    transfers and explicitly excluded transactions.

    Args:
        start_date: Optional start date (YYYY-MM-DD). Inclusive.
        end_date: Optional end date (YYYY-MM-DD). Inclusive.

    Returns:
        JSON object with total_expense, total_income, net, count,
        unclassified_count fields.
    """
    db = _get_session()
    try:
        q = _base_expense_query(db)
        q = _apply_date_filters(q, start_date, end_date)
        txns = q.all()

        total_expense = sum(t.amount for t in txns if t.amount < 0)
        total_income = sum(t.amount for t in txns if t.amount > 0)
        unclassified = sum(1 for t in txns if t.category_source == "unclassified")

        return _serialize(
            {
                "start_date": start_date,
                "end_date": end_date,
                "total_expense": round(total_expense, 2),
                "total_income": round(total_income, 2),
                "net": round(total_expense + total_income, 2),
                "transaction_count": len(txns),
                "unclassified_count": unclassified,
            }
        )
    finally:
        db.close()


@mcp.tool()
def get_category_breakdown(
    start_date: str | None = None,
    end_date: str | None = None,
    top_n: int = 10,
) -> str:
    """Get spending breakdown by category for a date range.

    Shows the top N categories ranked by total spending (absolute value).
    Each entry includes total amount, transaction count, and percentage
    of total spending. Only expense transactions (amount < 0) are included.

    Args:
        start_date: Optional start date (YYYY-MM-DD). Inclusive.
        end_date: Optional end date (YYYY-MM-DD). Inclusive.
        top_n: Number of top categories to return (default 10, max 50).

    Returns:
        JSON object with categories array and total_expense.
    """
    db = _get_session()
    try:
        top_n = min(top_n, 50)
        q = _base_expense_query(db)
        q = _apply_date_filters(q, start_date, end_date)
        # Only expenses
        q = q.filter(Transaction.amount < 0)
        txns = q.all()

        # Group by category
        cat_totals: dict[int | None, dict] = {}
        for t in txns:
            cid = t.category_id
            if cid not in cat_totals:
                cat_totals[cid] = {"amount": 0.0, "count": 0}
            cat_totals[cid]["amount"] += t.amount
            cat_totals[cid]["count"] += 1

        # Resolve category names
        cat_names = {
            c.id: c.name for c in db.query(Category).all()
        }

        total_expense = sum(v["amount"] for v in cat_totals.values())

        # Sort by absolute amount descending
        sorted_cats = sorted(cat_totals.items(), key=lambda x: x[1]["amount"])
        result = []
        for cid, data in sorted_cats[:top_n]:
            pct = (data["amount"] / total_expense * 100) if total_expense else 0
            result.append(
                {
                    "category": cat_names.get(cid, "Uncategorized"),
                    "category_id": cid,
                    "total_amount": round(data["amount"], 2),
                    "transaction_count": data["count"],
                    "percentage": round(pct, 1),
                }
            )

        return _serialize(
            {
                "start_date": start_date,
                "end_date": end_date,
                "total_expense": round(total_expense, 2),
                "categories": result,
            }
        )
    finally:
        db.close()


@mcp.tool()
def get_monthly_trend(
    start_date: str | None = None,
    end_date: str | None = None,
    category_name: str | None = None,
) -> str:
    """Get monthly spending totals over time.

    Shows spending aggregated by month (YYYY-MM). Optionally filter to
    a specific category by name. Only expense transactions (amount < 0)
    are included.

    Args:
        start_date: Optional start date (YYYY-MM-DD). Inclusive.
        end_date: Optional end date (YYYY-MM-DD). Inclusive.
        category_name: Optional category name to filter by (case-insensitive).

    Returns:
        JSON object with months array [{month, total_amount, count}].
    """
    db = _get_session()
    try:
        q = _base_expense_query(db)
        q = _apply_date_filters(q, start_date, end_date)
        q = q.filter(Transaction.amount < 0)

        # Optional category filter
        if category_name:
            cat = (
                db.query(Category)
                .filter(func.lower(Category.name) == category_name.lower())
                .first()
            )
            if not cat:
                return _serialize({"error": f"Category '{category_name}' not found"})
            q = q.filter(Transaction.category_id == cat.id)

        txns = q.all()

        # Group by month
        monthly: dict[str, dict] = {}
        for t in txns:
            d = t.effective_date or t.raw_date
            month_key = d.strftime("%Y-%m")
            if month_key not in monthly:
                monthly[month_key] = {"amount": 0.0, "count": 0}
            monthly[month_key]["amount"] += t.amount
            monthly[month_key]["count"] += 1

        result = [
            {
                "month": k,
                "total_amount": round(v["amount"], 2),
                "transaction_count": v["count"],
            }
            for k, v in sorted(monthly.items())
        ]

        return _serialize(
            {
                "category_filter": category_name,
                "months": result,
            }
        )
    finally:
        db.close()


@mcp.tool()
def get_budget_status(month: str | None = None) -> str:
    """Check budget vs actual spending for a given month.

    Shows each budgeted category with its monthly limit, actual spending,
    remaining amount, and percentage used.

    Args:
        month: Month to check in YYYY-MM format. Defaults to current month.

    Returns:
        JSON object with budgets array and overall totals.
    """
    db = _get_session()
    try:
        # Determine month range
        if month:
            year, mon = map(int, month.split("-"))
        else:
            today = date.today()
            year, mon = today.year, today.month
            month = f"{year:04d}-{mon:02d}"

        start = date(year, mon, 1)
        # End of month
        if mon == 12:
            end = date(year + 1, 1, 1)
        else:
            end = date(year, mon + 1, 1)

        # Get all budgets with category names
        budgets = db.query(Budget, Category).join(Category).all()
        if not budgets:
            return _serialize({"month": month, "message": "No budgets configured", "budgets": []})

        transfer_ids = _get_transfer_category_ids(db)
        result = []
        total_limit = 0.0
        total_spent = 0.0

        for budget, cat in budgets:
            # Sum spending for this category in the month
            q = (
                db.query(func.coalesce(func.sum(Transaction.amount), 0))
                .filter(
                    Transaction.category_id == cat.id,
                    Transaction.is_excluded.is_(False),
                    Transaction.amount < 0,
                    func.coalesce(Transaction.effective_date, Transaction.raw_date) >= start,
                    func.coalesce(Transaction.effective_date, Transaction.raw_date) < end,
                )
            )
            if transfer_ids:
                q = q.filter(~Transaction.category_id.in_(transfer_ids))
            spent = abs(q.scalar() or 0)

            remaining = budget.monthly_limit - spent
            pct = (spent / budget.monthly_limit * 100) if budget.monthly_limit else 0

            result.append(
                {
                    "category": cat.name,
                    "monthly_limit": budget.monthly_limit,
                    "spent": round(spent, 2),
                    "remaining": round(remaining, 2),
                    "percentage_used": round(pct, 1),
                    "over_budget": spent > budget.monthly_limit,
                }
            )
            total_limit += budget.monthly_limit
            total_spent += spent

        # Sort by percentage used descending (most concerning first)
        result.sort(key=lambda x: x["percentage_used"], reverse=True)

        return _serialize(
            {
                "month": month,
                "total_budget": round(total_limit, 2),
                "total_spent": round(total_spent, 2),
                "total_remaining": round(total_limit - total_spent, 2),
                "budgets": result,
            }
        )
    finally:
        db.close()


@mcp.tool()
def search_transactions(
    search: str | None = None,
    start_date: str | None = None,
    end_date: str | None = None,
    category_name: str | None = None,
    unclassified_only: bool = False,
    limit: int = 50,
) -> str:
    """Search for transactions by text, date, and/or category.

    Searches both raw_description and cleaned description fields. Results
    are ordered by date descending (most recent first).

    Args:
        search: Optional text to search in transaction descriptions (case-insensitive).
        start_date: Optional start date (YYYY-MM-DD). Inclusive.
        end_date: Optional end date (YYYY-MM-DD). Inclusive.
        category_name: Optional category name to filter by (case-insensitive).
        unclassified_only: If True, only return unclassified transactions.
        limit: Maximum number of transactions to return (default 50, max 200).

    Returns:
        JSON object with transactions array, total matching count, and total amount.
    """
    db = _get_session()
    try:
        limit = min(limit, 200)
        q = db.query(Transaction)

        # Text search
        if search:
            pattern = f"%{search}%"
            q = q.filter(
                Transaction.raw_description.ilike(pattern)
                | Transaction.description.ilike(pattern)
            )

        # Date filters
        q = _apply_date_filters(q, start_date, end_date)

        # Category filter by name
        if category_name:
            cat = (
                db.query(Category)
                .filter(func.lower(Category.name) == category_name.lower())
                .first()
            )
            if not cat:
                return _serialize({"error": f"Category '{category_name}' not found"})
            q = q.filter(Transaction.category_id == cat.id)

        # Unclassified filter
        if unclassified_only:
            q = q.filter(Transaction.category_source == "unclassified")

        # Get total count and sum before applying limit
        all_matching = q.all()
        total_count = len(all_matching)
        total_amount = round(sum(t.amount for t in all_matching), 2)

        # Order and limit
        q = q.order_by(
            func.coalesce(Transaction.effective_date, Transaction.raw_date).desc()
        ).limit(limit)
        txns = q.all()

        # Resolve category names for results
        cat_names = {c.id: c.name for c in db.query(Category).all()}

        result = [
            {
                "id": t.id,
                "date": str(t.effective_date or t.raw_date),
                "description": t.description or t.raw_description,
                "raw_description": t.raw_description,
                "amount": t.amount,
                "category": cat_names.get(t.category_id, "Uncategorized"),
                "category_source": t.category_source,
                "is_excluded": t.is_excluded,
                "notes": t.notes,
            }
            for t in txns
        ]

        return _serialize(
            {
                "total_count": total_count,
                "total_amount": total_amount,
                "showing": len(result),
                "transactions": result,
            }
        )
    finally:
        db.close()


@mcp.tool()
def get_merchant_spend(
    merchant_name: str,
    start_date: str | None = None,
    end_date: str | None = None,
) -> str:
    """Get total spending at a specific merchant.

    First looks up the merchant in the merchants table and uses its
    pattern aliases for matching. If not found there, falls back to
    searching raw_description directly.

    Args:
        merchant_name: Merchant name to search for (case-insensitive).
        start_date: Optional start date (YYYY-MM-DD). Inclusive.
        end_date: Optional end date (YYYY-MM-DD). Inclusive.

    Returns:
        JSON object with total spent, transaction count, date range,
        and the 10 most recent transactions.
    """
    db = _get_session()
    try:
        # Try to find merchant in merchants table
        merchant = (
            db.query(Merchant)
            .filter(func.lower(Merchant.name) == merchant_name.lower())
            .first()
        )

        q = db.query(Transaction)

        if merchant:
            # Use merchant patterns for matching
            patterns = json.loads(merchant.patterns) if merchant.patterns else []
            patterns.append(merchant.name)  # Always include canonical name
            # Match any pattern against raw_description
            conditions = [
                Transaction.raw_description.ilike(f"%{p}%") for p in patterns
            ]
            from sqlalchemy import or_

            q = q.filter(or_(*conditions))
        else:
            # Fallback: direct search on raw_description
            q = q.filter(Transaction.raw_description.ilike(f"%{merchant_name}%"))

        q = _apply_date_filters(q, start_date, end_date)
        txns = q.order_by(
            func.coalesce(Transaction.effective_date, Transaction.raw_date).desc()
        ).all()

        if not txns:
            return _serialize(
                {
                    "merchant": merchant_name,
                    "message": "No transactions found",
                    "total_spent": 0,
                    "transaction_count": 0,
                }
            )

        total = round(sum(t.amount for t in txns), 2)
        dates = [t.effective_date or t.raw_date for t in txns]

        # Category names for recent transactions
        cat_names = {c.id: c.name for c in db.query(Category).all()}

        recent = [
            {
                "id": t.id,
                "date": str(t.effective_date or t.raw_date),
                "description": t.description or t.raw_description,
                "amount": t.amount,
                "category": cat_names.get(t.category_id, "Uncategorized"),
            }
            for t in txns[:10]
        ]

        return _serialize(
            {
                "merchant": merchant.name if merchant else merchant_name,
                "total_spent": total,
                "transaction_count": len(txns),
                "first_date": str(min(dates)),
                "last_date": str(max(dates)),
                "recent_transactions": recent,
            }
        )
    finally:
        db.close()


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------
if __name__ == "__main__":
    mcp.run(transport="stdio")
