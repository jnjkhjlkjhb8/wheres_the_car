# Build context: repo root
# docker build -f server/router/Dockerfile -t bus-router .
FROM golang:1.26.3-alpine3.23 AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY models/ ./models/
COPY server/router/ ./server/router/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o router ./server/router

FROM alpine:3.23
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /app/router .
EXPOSE 50051 8080
ENTRYPOINT ["./router"]
