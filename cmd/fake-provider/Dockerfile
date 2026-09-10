# Build stage
FROM golang:alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/fake-provider ./cmd/fake-provider

# Final minimal runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates wget curl

COPY --from=builder /bin/fake-provider /usr/local/bin/fake-provider

EXPOSE 8082 8089

USER nobody:nobody

ENTRYPOINT ["/usr/local/bin/fake-provider"]
