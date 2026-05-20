FROM golang:1.26.3-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
WORKDIR /src/server/router
RUN go build -o /out/router .

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/router /usr/local/bin/router

ENTRYPOINT ["router"]
