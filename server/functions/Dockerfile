FROM golang:1.26.3-alpine3.23 AS builder
RUN sudo apk add --no-cache protobuf-dev protoc git
WORKDIR /app
COPY go.mod go.sum ./
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go mod download
COPY models/ ./models/
COPY server/ ./server/
RUN protoc --go_out=. --go_opt=paths=source_relative models/*.proto
RUN CGO_ENABLED=0 GOOS=linux go build -o handler ./server/functions

FROM alpine:3.23
RUN apk add --no-cache ca-certificates
WORKDIR /root/
COPY --from=builder /app/handler .
EXPOSE 8080
ENTRYPOINT ["./handler"]
