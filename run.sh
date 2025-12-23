#!/bin/bash
# Start the accounting tool (backend + frontend)

cd "$(dirname "$0")"

# Activate conda environment
eval "$(conda shell.bash hook)"
conda activate accounting-tool

# Initialize database if needed
if [ ! -f "data/accounting.db" ]; then
    echo "Initializing database..."
    python scripts/init_db.py
fi

# Start backend
echo "Starting API server at http://localhost:8000"
uvicorn backend.main:app --host 127.0.0.1 --port 8000 &
BACKEND_PID=$!

# Start frontend
echo "Starting frontend at http://localhost:5173"
cd frontend && npm run dev -- --host 127.0.0.1 &
FRONTEND_PID=$!

echo ""
echo "==================================="
echo "  Accounting Tool is running!"
echo "  Frontend: http://localhost:5173"
echo "  API:      http://localhost:8000"
echo "  Press Ctrl+C to stop"
echo "==================================="

# Wait for Ctrl+C
trap "kill $BACKEND_PID $FRONTEND_PID 2>/dev/null" EXIT
wait
