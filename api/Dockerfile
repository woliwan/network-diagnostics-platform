FROM golang:1.20-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /nettools ./...

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /nettools /nettools
EXPOSE 8080
ENTRYPOINT ["/nettools"]