# Название бинарного файла и путь сборки
APP_NAME := namaz-bot
BUILD_DIR := bin
MAIN_FILE := main.go

.PHONY: all build run test test-coverage clean fmt tidy check help

# Цель по умолчанию: вывод справки
all: help

## build: Собрать исполняемый бинарный файл
build:
	@echo "==> Сборка бинарного файла $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-w -s" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)
	@echo "==> Готово: $(BUILD_DIR)/$(APP_NAME)"

## run: Запустить приложение локально
run:
	@echo "==> Запуск бота..."
	go run $(MAIN_FILE)

## test: Запустить все модульные тесты
test:
	@echo "==> Запуск тестов..."
	go test -v -race ./...

## test-coverage: Запустить тесты с генерацией отчета о покрытии
test-coverage:
	@echo "==> Проверка покрытия тестами..."
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@rm -f coverage.out

## fmt: Отформатировать исходный код (gofmt)
fmt:
	@echo "==> Форматирование кода..."
	go fmt ./...

## tidy: Обновить и очистить зависимости в go.mod / go.sum
tidy:
	@echo "==> Обновление зависимостей..."
	go mod tidy

## check: Проверить форматирование и запустить тесты
check: fmt tidy test

## clean: Удалить временные файлы и собранные бинарники
clean:
	@echo "==> Очистка..."
	@rm -rf $(BUILD_DIR)
	@rm -f test_*.db coverage.out
	@echo "==> Готово."

## help: Показать список доступных команд
help:
	@echo "Использование: make [команда]"
	@echo ""
	@echo "Доступные команды:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
