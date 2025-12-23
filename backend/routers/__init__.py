"""API routers."""
from .transactions import router as transactions_router
from .statements import router as statements_router
from .categories import router as categories_router
from .rules import router as rules_router
from .receipts import router as receipts_router

__all__ = [
    "transactions_router",
    "statements_router",
    "categories_router",
    "rules_router",
    "receipts_router",
]
