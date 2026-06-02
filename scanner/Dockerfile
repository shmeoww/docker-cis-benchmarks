# ─── Стадия 1: сборка ────────────────────────────────────────────────────────
# Используем официальный образ Go для компиляции.
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Сначала копируем только файлы зависимостей — это отдельный слой кэша.
# Если go.mod/go.sum не изменились, Docker пропустит go mod download при следующей сборке.
COPY go.mod go.sum ./
RUN go mod download

# Теперь копируем исходники и собираем статически слинкованный бинарник.
# CGO_ENABLED=0 — отключаем CGO (нет зависимости от системных библиотек).
# GOOS=linux  — целевая ОС Linux (контейнер).
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o scanner .


# ─── Стадия 2: запуск ────────────────────────────────────────────────────────
# Минимальный образ — только Alpine и наш бинарник.
# Размер итогового образа: ~15 МБ вместо ~300 МБ с полным Go.
FROM alpine:3.21

# ca-certificates нужен для HTTPS-запросов (Trivy и т.п.)
RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /app/scanner .

EXPOSE 8000
CMD ["./scanner"]