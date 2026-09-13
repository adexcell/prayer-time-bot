package bot

import (
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"namaz-time-bot/internal/api"
	"namaz-time-bot/internal/storage"
)

type Bot struct {
	api         *tgbotapi.BotAPI
	client      *api.Client
	storage     *storage.Storage
	defaultCity string
}

func New(token, defaultCity string, client *api.Client, store *storage.Storage) (*Bot, error) {
	botAPI, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	log.Printf("Авторизован аккаунт бота: %s", botAPI.Self.UserName)

	return &Bot{
		api:         botAPI,
		client:      client,
		storage:     store,
		defaultCity: defaultCity,
	}, nil
}

// Start запускает цикл прослушивания входящих сообщений
func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		if update.CallbackQuery != nil {
			b.handleCallback(update.CallbackQuery)
			continue
		}

		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		text := update.Message.Text

		switch text {
		case "/start":
			b.handleStart(chatID)
		case "/today", "🕌 Расписание на сегодня":
			b.handleToday(chatID)
		case "/city", "🏙 Выбрать город":
			b.handleChooseLocation(chatID, true, 1, 0)
		case "/settings", "⚙️ Настройки":
			b.handleSettings(chatID, 0)
		case "/subscribe", "🔔 Подписаться на рассылку":
			b.handleSubscribe(chatID)
		case "/unsubscribe", "🔕 Отписаться":
			b.handleUnsubscribe(chatID)
		default:
			b.sendMessage(chatID, "Используйте меню или команды:\n/today — Расписание на сегодня\n/city — Выбрать город или район РБ\n/settings — Настройки уведомлений\n/subscribe — Подписаться на рассылку\n/unsubscribe — Отписаться")
		}
	}
}

const locationPageSize = 12

func (b *Bot) handleChooseLocation(chatID int64, isCityTab bool, page int, messageID int) {
	if page < 1 {
		page = 1
	}

	var items []api.CityInfo
	if isCityTab {
		items = api.GetCitiesOnly()
	} else {
		items = api.GetDistrictsOnly()
	}

	totalItems := len(items)
	totalPages := (totalItems + locationPageSize - 1) / locationPageSize
	if page > totalPages {
		page = totalPages
	}

	startIdx := (page - 1) * locationPageSize
	endIdx := startIdx + locationPageSize
	if endIdx > totalItems {
		endIdx = totalItems
	}

	currentCity, _ := b.storage.GetUserCity(chatID, b.defaultCity)

	tabName := "🌆 Города"
	if !isCityTab {
		tabName = "🏡 Районы"
	}

	safeCurrentCity := escapeMarkdown(currentCity)
	safeTabName := escapeMarkdown(tabName)

	text := fmt.Sprintf(
		"🏙 *Выберите населенный пункт Республики Башкортостан:*\n\n"+
			"Вкладка: %s\n"+
			"📍 Текущий выбор: %s\n"+
			"📄 Страница: %d из %d",
		safeTabName, safeCurrentCity, page, totalPages,
	)

	var rows [][]tgbotapi.InlineKeyboardButton

	// 1. Вкладки категорий
	citiesTabLabel := "🌆 Города (21)"
	districtsTabLabel := "🏡 Районы (40)"
	if isCityTab {
		citiesTabLabel = "🔹 🌆 Города (21)"
	} else {
		districtsTabLabel = "🔹 🏡 Районы (40)"
	}

	tabRow := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(citiesTabLabel, "tab_cities:1"),
		tgbotapi.NewInlineKeyboardButtonData(districtsTabLabel, "tab_districts:1"),
	}
	rows = append(rows, tabRow)

	// 2. Кнопки городов/районов
	pageItems := items[startIdx:endIdx]
	var currentRow []tgbotapi.InlineKeyboardButton

	for _, item := range pageItems {
		btnText := item.DisplayName
		if item.DisplayName == currentCity || item.CleanName == currentCity {
			btnText = "✓ " + item.DisplayName
		}
		btn := tgbotapi.NewInlineKeyboardButtonData(btnText, fmt.Sprintf("set_city:%d", item.ID))
		currentRow = append(currentRow, btn)

		if len(currentRow) == 2 {
			rows = append(rows, currentRow)
			currentRow = []tgbotapi.InlineKeyboardButton{}
		}
	}
	if len(currentRow) > 0 {
		rows = append(rows, currentRow)
	}

	// 3. Строка пагинации
	pagePrefix := "tab_cities"
	if !isCityTab {
		pagePrefix = "tab_districts"
	}

	var navRow []tgbotapi.InlineKeyboardButton
	if page > 1 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("%s:%d", pagePrefix, page-1)))
	} else {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(" ", "noop"))
	}

	navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d / %d", page, totalPages), "noop"))

	if page < totalPages {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("Вперед ➡️", fmt.Sprintf("%s:%d", pagePrefix, page+1)))
	} else {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(" ", "noop"))
	}

	rows = append(rows, navRow)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	if messageID > 0 {
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
		editMsg.ParseMode = "Markdown"
		editMsg.ReplyMarkup = &keyboard
		if _, err := b.api.Send(editMsg); err != nil {
			if !strings.Contains(err.Error(), "message is not modified") {
				log.Printf("Ошибка редактирования сообщения в handleChooseLocation: %v", err)
			}
		}
	} else {
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard
		if _, err := b.api.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения в handleChooseLocation: %v", err)
		}
	}
}

func escapeMarkdown(s string) string {
	r := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"`", "\\`",
		"[", "\\[",
	)
	return r.Replace(s)
}

func (b *Bot) handleSettings(chatID int64, messageID int) {
	u, _ := b.storage.GetUser(chatID)

	selectedTime := "06:00"
	notify15min := true
	notifyAtTime := true

	if u != nil {
		if u.DailyScheduleTime != "" {
			selectedTime = u.DailyScheduleTime
		}
		notify15min = u.Notify15Min
		notifyAtTime = u.NotifyAtTime
	}

	currentCity, _ := b.storage.GetUserCity(chatID, b.defaultCity)

	text := fmt.Sprintf(
		"⚙️ *Настройки рассылки и напоминаний*\n\n"+
			"📍 *Текущий город/район:* %s\n"+
			"⏰ *Время утренней рассылки:* %s\n\n"+
			"Выберите время рассылки или включите/выключите нужные напоминания:",
		currentCity, selectedTime,
	)

	times := []string{"05:00", "06:00", "07:00", "08:00", "09:00", "10:00"}
	var timeRow1 []tgbotapi.InlineKeyboardButton
	var timeRow2 []tgbotapi.InlineKeyboardButton

	for i, t := range times {
		label := t
		if t == selectedTime {
			label = "✓ " + t
		}
		btn := tgbotapi.NewInlineKeyboardButtonData(label, "set_time:"+t)
		if i < 3 {
			timeRow1 = append(timeRow1, btn)
		} else {
			timeRow2 = append(timeRow2, btn)
		}
	}

	label15 := "⏳ За 15 мин: [ ]"
	if notify15min {
		label15 = "⏳ За 15 мин: [✓]"
	}

	labelAtTime := "🔔 В момент намаза: [ ]"
	if notifyAtTime {
		labelAtTime = "🔔 В момент намаза: [✓]"
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		timeRow1,
		timeRow2,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label15, "toggle_15min"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(labelAtTime, "toggle_attime"),
		),
	)

	if messageID > 0 {
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
		editMsg.ParseMode = "Markdown"
		editMsg.ReplyMarkup = &keyboard
		b.api.Send(editMsg)
	} else {
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard
		b.api.Send(msg)
	}
}

func (b *Bot) handleCallback(cb *tgbotapi.CallbackQuery) {
	chatID := cb.Message.Chat.ID
	messageID := cb.Message.MessageID
	data := cb.Data

	if data == "noop" {
		b.answerCallback(cb.ID, "")
		return
	}

	if strings.HasPrefix(data, "tab_cities:") {
		pageStr := strings.TrimPrefix(data, "tab_cities:")
		var page int
		fmt.Sscanf(pageStr, "%d", &page)
		b.handleChooseLocation(chatID, true, page, messageID)
		b.answerCallback(cb.ID, "")
		return
	}

	if strings.HasPrefix(data, "tab_districts:") {
		pageStr := strings.TrimPrefix(data, "tab_districts:")
		var page int
		fmt.Sscanf(pageStr, "%d", &page)
		b.handleChooseLocation(chatID, false, page, messageID)
		b.answerCallback(cb.ID, "")
		return
	}

	if strings.HasPrefix(data, "set_city:") {
		val := strings.TrimPrefix(data, "set_city:")
		cityName := val

		var cityID int
		if _, err := fmt.Sscanf(val, "%d", &cityID); err == nil && cityID > 0 {
			if cityInfo, ok := api.GetCityByID(cityID); ok {
				cityName = cityInfo.DisplayName
			}
		}

		_ = b.storage.SetUserCity(chatID, cityName)
		b.answerCallback(cb.ID, "Выбрано: "+cityName)

		// Отправляем расписание для выбранного города/района
		msgText := fmt.Sprintf("✅ *Выбран населенный пункт: %s*\n\nНиже расписание намаза на сегодня:", cityName)
		b.sendMessage(chatID, msgText)
		b.sendTodayForCity(chatID, cityName)
		return
	}

	if strings.HasPrefix(data, "set_time:") {
		timeStr := strings.TrimPrefix(data, "set_time:")
		_ = b.storage.UpdateDailyTime(chatID, timeStr)
		b.answerCallback(cb.ID, "Время утренней рассылки: "+timeStr)
		b.handleSettings(chatID, messageID)
		return
	}

	switch data {
	case "toggle_15min":
		_ = b.storage.ToggleNotify15min(chatID)
		b.answerCallback(cb.ID, "Настройка напоминания за 15 мин изменена")
		b.handleSettings(chatID, messageID)
	case "toggle_attime":
		_ = b.storage.ToggleNotifyAtTime(chatID)
		b.answerCallback(cb.ID, "Настройка напоминания в момент намаза изменена")
		b.handleSettings(chatID, messageID)
	}
}

func (b *Bot) answerCallback(callbackID, text string) {
	callback := tgbotapi.NewCallback(callbackID, text)
	b.api.Request(callback)
}

func (b *Bot) handleStart(chatID int64) {
	city, _ := b.storage.GetUserCity(chatID, b.defaultCity)
	msgText := fmt.Sprintf(
		"Ассаляму алейкум! 🖐\n\n"+
			"Этот бот показывает официальное расписание намаза от ДУМ РБ для Республики Башкортостан.\n\n"+
			"📍 Ваш текущий город: *%s*\n\n"+
			"Выберите действие в меню ниже:",
		city,
	)
	b.sendMessageWithKeyboard(chatID, msgText)
}

func (b *Bot) handleToday(chatID int64) {
	city, _ := b.storage.GetUserCity(chatID, b.defaultCity)
	b.sendTodayForCity(chatID, city)
}

func (b *Bot) sendTodayForCity(chatID int64, city string) {
	now := time.Now()
	item, err := b.client.FetchPrayerTimes(city, now)
	if err != nil {
		log.Printf("Ошибка получения времени намаза для %s (chatID %d): %v", city, chatID, err)
		b.sendMessage(chatID, fmt.Sprintf("К сожалению, не удалось получить расписание для города %s с сервера ДУМ РБ.", city))
		return
	}

	msgText := api.FormatMessage(item, city, now)
	b.sendMessage(chatID, msgText)
}

func (b *Bot) handleSubscribe(chatID int64) {
	city, _ := b.storage.GetUserCity(chatID, b.defaultCity)
	if err := b.storage.Subscribe(chatID, city); err != nil {
		log.Printf("Ошибка подписки chatID %d: %v", chatID, err)
		b.sendMessage(chatID, "Произошла ошибка при оформлении подписки.")
		return
	}
	b.sendMessage(chatID, fmt.Sprintf("✅ Вы успешно подписались на ежедневную утреннюю рассылку расписания намаза для города *%s*!", city))
}

func (b *Bot) handleUnsubscribe(chatID int64) {
	if err := b.storage.Unsubscribe(chatID); err != nil {
		log.Printf("Ошибка отписки chatID %d: %v", chatID, err)
		b.sendMessage(chatID, "Произошла ошибка при отписке.")
		return
	}
	b.sendMessage(chatID, "🔕 Вы отписались от ежедневной рассылки.")
}

func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}

func (b *Bot) sendMessageWithKeyboard(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🕌 Расписание на сегодня"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🏙 Выбрать город"),
			tgbotapi.NewKeyboardButton("⚙️ Настройки"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔔 Подписаться на рассылку"),
			tgbotapi.NewKeyboardButton("🔕 Отписаться"),
		),
	)
	keyboard.ResizeKeyboard = true
	msg.ReplyMarkup = keyboard

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения с клавиатурой: %v", err)
	}
}

// SendToChat используется планировщиком для рассылки сообщений
func (b *Bot) SendToChat(chatID int64, text string) {
	b.sendMessage(chatID, text)
}

