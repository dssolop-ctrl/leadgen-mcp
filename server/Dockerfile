# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go clean -cache && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o leadgen-mcp .

# Runtime stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/leadgen-mcp .

EXPOSE 8080

ENTRYPOINT ["./leadgen-mcp"]
CMD ["-config", "/app/config.yaml"]
