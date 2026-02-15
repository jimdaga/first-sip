# Multi-stage build for First Sip CrewAI Sidecar
FROM python:3.12-slim AS builder

# Install uv for fast dependency installation
RUN pip install --no-cache-dir uv

# Copy project configuration
WORKDIR /app/sidecar
COPY pyproject.toml ./

# Install dependencies using uv
RUN uv pip install --system .

# Runtime stage
FROM python:3.12-slim

# Copy installed packages from builder
COPY --from=builder /usr/local/lib/python3.12/site-packages /usr/local/lib/python3.12/site-packages
COPY --from=builder /usr/local/bin /usr/local/bin

# Set working directory
WORKDIR /app/sidecar

# Copy sidecar source code
COPY *.py ./

# Copy plugins directory (crew definitions)
COPY ../plugins /app/plugins

# Expose FastAPI port
EXPOSE 8000

# Labels
LABEL maintainer="First Sip Team"
LABEL description="CrewAI workflow execution sidecar for First Sip plugin architecture"

# Run uvicorn with multiple workers
# Use 2 workers (not too many - CrewAI workflows are CPU-bound per execution)
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000", "--workers", "2"]
