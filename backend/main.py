"""FastAPI application entry point."""
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from .database import init_db
from .routers import (
    transactions_router,
    statements_router,
    categories_router,
    rules_router,
)

app = FastAPI(
    title="Personal Accounting System",
    description="Personal finance tracking with bank statement imports",
    version="0.1.0",
)

# CORS for frontend
app.add_middleware(
    CORSMiddleware,
    allow_origins=["http://localhost:5173", "http://127.0.0.1:5173"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Include routers
app.include_router(transactions_router)
app.include_router(statements_router)
app.include_router(categories_router)
app.include_router(rules_router)


@app.on_event("startup")
def on_startup():
    """Initialize database on startup."""
    init_db()


@app.get("/")
def root():
    """Root endpoint."""
    return {"status": "ok", "message": "Personal Accounting System API"}


@app.get("/health")
def health():
    """Health check endpoint."""
    return {"status": "healthy"}
