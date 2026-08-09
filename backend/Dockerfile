FROM golang:1.25-alpine AS build
WORKDIR /app

ENV GOTOOLCHAIN=auto

# Копируем весь исходный код и go.mod целиком
COPY . .

# Автоматически создаем go.sum и скачиваем зависимости внутри контейнера с Go 1.25
RUN go mod tidy

# Собираем приложение
RUN go build -o /app/server ./cmd/api

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=build /app/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]