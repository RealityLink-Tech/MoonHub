FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /relay ./cmd/relay/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /relay /usr/local/bin/relay
EXPOSE 8081
CMD ["relay"]
