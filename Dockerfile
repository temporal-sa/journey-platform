# Multi-stage Dockerfile for Journey Platform services
FROM golang:alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG SERVICE=fake-provider
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/service ./cmd/${SERVICE}

# Final minimal runtime image
FROM alpine:3.19

RUN apk add --no-cache ca-certificates wget curl

COPY --from=builder /bin/service /usr/local/bin/service

EXPOSE 8080 8082 8084 8085 8087 8088 8089

USER nobody:nobody

ENTRYPOINT ["/usr/local/bin/service"]
