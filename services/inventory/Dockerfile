FROM golang:1.21 AS builder

WORKDIR /app
COPY . .

# Fetch dependencies and build
RUN go get github.com/lib/pq google.golang.org/grpc@v1.60.1 google.golang.org/protobuf@v1.33.0
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /service ./services/inventory/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /service .
EXPOSE 50052
CMD ["./service"]
