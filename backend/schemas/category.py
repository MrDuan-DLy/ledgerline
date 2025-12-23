"""Category schemas."""
from pydantic import BaseModel, Field


class CategoryBase(BaseModel):
    """Base category fields."""
    name: str
    parent_id: int | None = None
    icon: str | None = None
    color: str | None = None
    is_expense: bool = True


class CategoryCreate(CategoryBase):
    """Fields for creating a category."""
    pass


class CategoryResponse(CategoryBase):
    """Category response."""
    id: int
    children: list["CategoryResponse"] = Field(default_factory=list)

    class Config:
        from_attributes = True


# For recursive model
CategoryResponse.model_rebuild()
