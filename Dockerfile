# Stage 1: Build Go backend
FROM golang:1.24-alpine AS backend-builder

WORKDIR /build

COPY backend-go/go.mod backend-go/go.sum ./
RUN go mod download

COPY backend-go/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

# Stage 2: Build frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /build

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ .
RUN npm run build

# Stage 3: Runtime
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=backend-builder /server ./server
COPY --from=frontend-builder /build/dist ./static

COPY configs/examples/ ./configs/examples/

RUN mkdir -p /app/data && \
    adduser -D -u 1001 app && chown -R app:app /app

USER app

EXPOSE 8000

VOLUME ["/app/data"]

ENTRYPOINT ["./server"]
