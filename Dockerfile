FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/stock-service ./cmd/stock-service
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/invoicing-service ./cmd/invoicing-service

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/stock-service /app/stock-service
COPY --from=builder /bin/invoicing-service /app/invoicing-service

ENV PORT=8080
EXPOSE 8080

CMD ["sh", "-c", "if [ \"$SERVICE_NAME\" = \"invoicing-service\" ]; then exec /app/invoicing-service; else exec /app/stock-service; fi"]
