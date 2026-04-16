# Build stage
FROM golang:1.23-alpine AS builder

RUN apk --no-cache add git

WORKDIR /app

COPY go.mod go.sum ./
RUN GOPROXY=direct go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o leadgen-mcp .

# Runtime stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates

WORKDIR /app
RUN mkdir -p /app/data
COPY --from=builder /app/leadgen-mcp .

EXPOSE 8080

# /app/data — data directory for SQLite DB and JSON seed file.
# Mount this as a bind mount (see docker-compose.yml at repo root):
#   ./server/data:/app/data
# We intentionally do NOT declare VOLUME here — that caused Docker to create
# anonymous volumes when `docker run` was invoked without -v, scattering data
# across throwaway volumes.

ENTRYPOINT ["./leadgen-mcp"]
CMD ["-config", "/app/config.yaml"]
