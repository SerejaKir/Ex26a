# Первая ступень - сборка
FROM golang:latest AS compiling_stage
LABEL version="v1.0"
LABEL maintainer="Sergey Kirichenko<sergeykirichenko@mail.ru>"

# Устанавливаем зависимости в виртуальное окружение

WORKDIR /app
COPY . .

# Компилируем с явным указанием пути для бинарника
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/ex26a .

# Вторая ступень - финальный образ
FROM alpine:latest

WORKDIR /app

# Копируем только виртуальное окружение из builder
COPY --from=compiling_stage /app/ex26a .

# Делаем файл исполняемым
RUN chmod +x /app/ex26a

# Команда для запуска приложения
ENTRYPOINT ["/app/ex26a"]
