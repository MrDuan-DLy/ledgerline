"""Rule schemas."""
from datetime import datetime
from pydantic import BaseModel


class RuleBase(BaseModel):
    """Base rule fields."""
    pattern: str
    pattern_type: str = "contains"  # 'contains', 'regex', 'exact'
    category_id: int
    priority: int = 0
    is_active: bool = True


class RuleCreate(RuleBase):
    """Fields for creating a rule."""
    created_from_txn_id: int | None = None


class RuleResponse(RuleBase):
    """Rule response."""
    id: int
    category_name: str | None = None
    created_at: datetime

    class Config:
        from_attributes = True
