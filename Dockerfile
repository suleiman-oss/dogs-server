# ── Stage 1: Build ────────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o dogs-server ./cmd/server

# ── Stage 2: Runtime ──────────────────────────────────────────────────────────
FROM alpine:3.19

WORKDIR /app

# Copy binary
COPY --from=builder /app/dogs-server .

# Copy seed data and frontend
COPY data/seed.json data/seed.json
COPY frontend/public/ frontend/public/

ENV PORT=3000
ENV DATA_FILE=data/dogs.json
ENV SEED_FILE=data/seed.json
ENV STATIC_DIR=frontend/public

EXPOSE 3000

CMD ["./dogs-server"]
