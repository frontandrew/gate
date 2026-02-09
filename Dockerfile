FROM golang:1.23-alpine AS builder

WORKDIR /app

# Устанавливаем зависимости для сборки
RUN apk add --no-cache git

# Копируем go.mod первым для кэширования зависимостей
COPY go.mod ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -o /gate-api ./cmd/api

# Финальный образ
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /gate-api .
COPY .env.example .env

EXPOSE 8080

CMD ["./gate-api"]
