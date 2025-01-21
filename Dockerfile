# Define the platform argument
ARG TARGETPLATFORM=linux/amd64

# Update to Go 1.23
FROM --platform=$TARGETPLATFORM golang:1.23-bullseye AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o main .

FROM --platform=linux/amd64 gcr.io/distroless/static-debian11

WORKDIR /app

COPY --from=builder /app/main .

EXPOSE 8080

# Add a healthcheck
#HEALTHCHECK --interval=30s --timeout=30s --start-period=5s --retries=3 \
#    CMD curl -f http://localhost:${PORT:-8080}/health || exit 1

CMD ["./main"]
