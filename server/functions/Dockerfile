# Build context: repo root
# docker build -f server/functions/Dockerfile -t bus-functions .

FROM golang:1.26.3-alpine3.23 AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
# Go pb stubs are committed — no protoc needed
COPY models/ ./models/
COPY server/functions/ ./server/functions/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o functions ./server/functions

FROM alpine:3.23
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /app/functions .
ENTRYPOINT ["./functions"]
