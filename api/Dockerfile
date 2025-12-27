FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install air for hot reloading (use compatible version)
RUN go install github.com/cosmtrek/air@v1.49.0

# Copy all source code first
COPY . .

# Download dependencies and tidy module (needs source code to see all imports)
RUN go mod download && go mod tidy

# Build the application
RUN go build -o main ./cmd/server/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .
COPY --from=builder /go/bin/air /usr/local/bin/air

# Copy source for development
COPY . .

EXPOSE 8080

CMD ["./main"]