"""Category API endpoints."""
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session

from ..database import get_db
from ..models import Category
from ..schemas import CategoryResponse, CategoryCreate

router = APIRouter(prefix="/api/categories", tags=["categories"])


@router.get("", response_model=list[CategoryResponse])
def list_categories(db: Session = Depends(get_db)):
    """List all categories (flat list)."""
    categories = db.query(Category).order_by(Category.name).all()
    return [
        CategoryResponse(
            id=c.id,
            name=c.name,
            parent_id=c.parent_id,
            icon=c.icon,
            color=c.color,
            is_expense=c.is_expense,
        )
        for c in categories
    ]


@router.get("/tree", response_model=list[CategoryResponse])
def list_categories_tree(db: Session = Depends(get_db)):
    """List categories as a tree structure."""
    categories = db.query(Category).all()

    # Build lookup
    by_id = {c.id: c for c in categories}

    # Build tree
    def build_node(cat: Category) -> CategoryResponse:
        children = [build_node(c) for c in categories if c.parent_id == cat.id]
        return CategoryResponse(
            id=cat.id,
            name=cat.name,
            parent_id=cat.parent_id,
            icon=cat.icon,
            color=cat.color,
            is_expense=cat.is_expense,
            children=children,
        )

    # Return only root categories (no parent)
    roots = [c for c in categories if c.parent_id is None]
    return [build_node(r) for r in roots]


@router.post("", response_model=CategoryResponse)
def create_category(data: CategoryCreate, db: Session = Depends(get_db)):
    """Create a new category."""
    # Check if name already exists
    existing = db.query(Category).filter(Category.name == data.name).first()
    if existing:
        raise HTTPException(status_code=400, detail="Category name already exists")

    # Verify parent if specified
    if data.parent_id:
        parent = db.query(Category).filter(Category.id == data.parent_id).first()
        if not parent:
            raise HTTPException(status_code=400, detail="Parent category not found")

    category = Category(**data.model_dump())
    db.add(category)
    db.commit()
    db.refresh(category)

    return CategoryResponse(
        id=category.id,
        name=category.name,
        parent_id=category.parent_id,
        icon=category.icon,
        color=category.color,
        is_expense=category.is_expense,
    )


@router.delete("/{category_id}")
def delete_category(category_id: int, db: Session = Depends(get_db)):
    """Delete a category (if no transactions are using it)."""
    category = db.query(Category).filter(Category.id == category_id).first()
    if not category:
        raise HTTPException(status_code=404, detail="Category not found")

    # Check if any transactions use this category
    if category.transactions:
        raise HTTPException(
            status_code=400,
            detail=f"Cannot delete: {len(category.transactions)} transactions use this category",
        )

    # Check for child categories
    if category.children:
        raise HTTPException(
            status_code=400,
            detail="Cannot delete: category has child categories",
        )

    db.delete(category)
    db.commit()
    return {"deleted": True}
