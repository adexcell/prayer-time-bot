package main

import (
	"log"

	"namaz-time-bot/config"
	"namaz-time-bot/internal/api"
	"namaz-time-bot/internal/bot"
	"namaz-time-bot/internal/scheduler"
	"namaz-time-bot/internal/storage"
)

func main() {
	log.Println("Запуск Namaz Time Bot...")

	// 1. Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфига: %v", err)
	}

	// 2. Инициализация SQLite хранилища
	store, err := storage.New("bot.db")
	if err != nil {
		log.Fatalf("Ошибка базы данных: %v", err)
	}
	defer store.Close()

	// 3. Инициализация клиента ДУМ РБ и привязка хранилища для учета корректировок
	apiClient := api.NewClient()
	apiClient.SetStorage(store)

	// 4. Инициализация бота
	adminIDs := cfg.GetAdminIDList()
	telegramBot, err := bot.New(cfg.TelegramToken, cfg.City, adminIDs, apiClient, store)
	if err != nil {
		log.Fatalf("Ошибка создания бота: %v", err)
	}

	// 5. Инициализация и запуск планировщика рассылки
	cronSched, err := scheduler.New(telegramBot, apiClient, store, cfg.City)
	if err != nil {
		log.Fatalf("Ошибка планировщика: %v", err)
	}
	cronSched.Start()
	defer cronSched.Stop()

	// 6. Запуск обработки сообщений бота
	log.Println("Бот успешно запущен и готов к работе!")
	telegramBot.Start()
}
