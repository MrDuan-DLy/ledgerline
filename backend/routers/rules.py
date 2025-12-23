"""Rule API endpoints."""
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session

from ..database import get_db
from ..models import Rule, Category
from ..schemas import RuleResponse, RuleCreate
from ..services import ClassifyService

router = APIRouter(prefix="/api/rules", tags=["rules"])


@router.get("", response_model=list[RuleResponse])
def list_rules(db: Session = Depends(get_db)):
    """List all classification rules."""
    rules = db.query(Rule).order_by(Rule.priority.desc(), Rule.pattern).all()
    return [
        RuleResponse(
            id=r.id,
            pattern=r.pattern,
            pattern_type=r.pattern_type,
            category_id=r.category_id,
            category_name=r.category.name if r.category else None,
            priority=r.priority,
            is_active=r.is_active,
            created_at=r.created_at,
        )
        for r in rules
    ]


@router.post("", response_model=RuleResponse)
def create_rule(data: RuleCreate, db: Session = Depends(get_db)):
    """Create a new classification rule."""
    # Verify category
    category = db.query(Category).filter(Category.id == data.category_id).first()
    if not category:
        raise HTTPException(status_code=400, detail="Category not found")

    rule = Rule(**data.model_dump())
    db.add(rule)
    db.commit()
    db.refresh(rule)

    return RuleResponse(
        id=rule.id,
        pattern=rule.pattern,
        pattern_type=rule.pattern_type,
        category_id=rule.category_id,
        category_name=category.name,
        priority=rule.priority,
        is_active=rule.is_active,
        created_at=rule.created_at,
    )


@router.delete("/{rule_id}")
def delete_rule(rule_id: int, db: Session = Depends(get_db)):
    """Delete a rule."""
    rule = db.query(Rule).filter(Rule.id == rule_id).first()
    if not rule:
        raise HTTPException(status_code=404, detail="Rule not found")

    db.delete(rule)
    db.commit()
    return {"deleted": True}


@router.patch("/{rule_id}/toggle")
def toggle_rule(rule_id: int, db: Session = Depends(get_db)):
    """Toggle rule active status."""
    rule = db.query(Rule).filter(Rule.id == rule_id).first()
    if not rule:
        raise HTTPException(status_code=404, detail="Rule not found")

    rule.is_active = not rule.is_active
    db.commit()
    return {"is_active": rule.is_active}


@router.post("/reclassify")
def reclassify_all(db: Session = Depends(get_db)):
    """Re-run all rules on non-manual transactions."""
    service = ClassifyService(db)
    updated = service.reclassify_all()
    return {"updated": updated}
