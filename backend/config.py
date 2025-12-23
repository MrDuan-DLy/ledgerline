"""Application configuration."""
from pathlib import Path
import os


def _load_dotenv(path: Path) -> None:
    """Load environment variables from a .env file if it exists."""
    if not path.exists():
        return
    for line in path.read_text().splitlines():
        stripped = line.strip()
        if not stripped or stripped.startswith("#") or "=" not in stripped:
            continue
        key, value = stripped.split("=", 1)
        key = key.strip()
        value = value.strip().strip('"').strip("'")
        os.environ.setdefault(key, value)

# Project paths
BASE_DIR = Path(__file__).resolve().parent.parent
DATA_DIR = BASE_DIR / "data"
UPLOADS_DIR = DATA_DIR / "uploads"
RECEIPTS_DIR = DATA_DIR / "receipts"

_load_dotenv(BASE_DIR / ".env")

# Database
DATABASE_URL = f"sqlite:///{DATA_DIR}/accounting.db"

# Ensure directories exist
DATA_DIR.mkdir(exist_ok=True)
UPLOADS_DIR.mkdir(exist_ok=True)
RECEIPTS_DIR.mkdir(exist_ok=True)
