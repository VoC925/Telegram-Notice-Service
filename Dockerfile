# базовый образ с alias builder 
FROM golang:1.22 AS builder
# рабочая директория
WORKDIR /usr/src
# копирование всех файлов и каталогов
COPY . .
# подгрузка зависимостей
RUN go mod tidy
# переменные окружения
ENV CGO_ENABLED 0
ENV GOOS linux
# build файла /usr/src/cmd/main.go в /usr/src/bin/app
RUN go build -o ./bin/app ./cmd/main.go
# второй этап сборки
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
# копирование двоичного файла в рабочую директорию из образа golang в образ alpine
COPY --from=builder /usr/src/bin/app .
COPY --from=builder /usr/src/config.yml .
COPY --from=builder /usr/src/app.log .
# Run
CMD ["./app"]