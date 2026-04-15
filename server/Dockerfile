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

# /app/data — persistent volume for SQLite databases (filters.db)
VOLUME ["/app/data"]

ENTRYPOINT ["./leadgen-mcp"]
CMD ["-config", "/app/config.yaml"]
