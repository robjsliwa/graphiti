FROM golang:1.25-alpine AS builder

RUN go install github.com/a-h/templ/cmd/templ@latest

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN templ generate
RUN CGO_ENABLED=0 go build -o /graphiti ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /graphiti /graphiti
COPY --from=builder /app/config /config
COPY --from=builder /app/migrations /migrations
COPY --from=builder /app/web/static /web/static
EXPOSE 8080
CMD ["/graphiti"]
