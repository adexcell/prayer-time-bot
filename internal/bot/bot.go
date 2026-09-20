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
	api            *tgbotapi.BotAPI
	client         *api.Client
	storage        *storage.Storage
	defaultCity    string
	configAdminIDs []int64
}

func New(token, defaultCity string, configAdminIDs []int64, client *api.Client, store *storage.Storage) (*Bot, error) {
	botAPI, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	log.Printf("Авторизован аккаунт бота: %s", botAPI.Self.UserName)

	return &Bot{
		api:            botAPI,
		client:         client,
		storage:        store,
		defaultCity:    defaultCity,
		configAdminIDs: configAdminIDs,
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

		if update.MyChatMember != nil {
			b.handleMyChatMember(update.MyChatMember)
			continue
		}

		if update.ChannelPost != nil {
			continue
		}

		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		fromID := int64(0)
		if update.Message.From != nil {
			fromID = update.Message.From.ID
		}
		text := strings.TrimSpace(update.Message.Text)

		// 1. Проверка пересланного сообщения из канала (для мгновенной настройки канала в ЛС)
		if chatID > 0 && update.Message.ForwardFromChat != nil && update.Message.ForwardFromChat.IsChannel() {
			fwdChat := update.Message.ForwardFromChat
			if fromID != 0 && b.isGroupAdmin(fwdChat.ID, fromID) {
				title := fwdChat.Title
				if title == "" {
					title = fmt.Sprintf("Канал %d", fwdChat.ID)
				}
				b.handleGroupSettings(chatID, fwdChat.ID, title, 0)
				continue
			} else {
				b.sendMessage(chatID, "⛔️ Вы не являетесь администратором пересланного канала или бот не назначен администратором в нём.")
				continue
			}
		}

		// 2. Проверка добавления бота в группу/канал
		if len(update.Message.NewChatMembers) > 0 {
			for _, newMember := range update.Message.NewChatMembers {
				if newMember.ID == b.api.Self.ID {
					b.handleGroupWelcome(chatID)
					break
				}
			}
			continue
		}

		if text == "" {
			continue
		}

		// 3. Нормализация команд (удаление @BotUsername)
		cmd := text
		botUsernameSuffix := "@" + strings.ToLower(b.api.Self.UserName)
		parts := strings.Fields(text)
		if len(parts) > 0 {
			rawCmd := parts[0]
			if strings.Contains(rawCmd, "@") {
				lowerRaw := strings.ToLower(rawCmd)
				if strings.HasSuffix(lowerRaw, botUsernameSuffix) {
					rawCmd = rawCmd[:len(rawCmd)-len(botUsernameSuffix)]
				}
			}
			cmd = rawCmd
		}

		switch {
		case cmd == "/start":
			b.handleStart(chatID, fromID, parts)
		case cmd == "/today" || text == "🕌 Расписание на сегодня":
			b.handleToday(chatID)
		case cmd == "/city" || text == "🏙 Выбрать город":
			if chatID < 0 {
				b.handleGroupSettingsRedirect(chatID, update.Message.MessageID)
			} else {
				b.handleChooseLocation(chatID, true, 1, 0)
			}
		case cmd == "/settings" || text == "⚙️ Настройки":
			if chatID < 0 {
				b.handleGroupSettingsRedirect(chatID, update.Message.MessageID)
			} else {
				b.handleSettings(chatID, 0)
			}
		case cmd == "/channel":
			b.handleChannelCommand(chatID, fromID, parts)
		case cmd == "/subscribe" || text == "🔔 Подписаться на рассылку":
			if chatID < 0 && !b.isGroupAdmin(chatID, fromID) {
				b.sendMessage(chatID, "⛔️ Включать рассылку для группы могут только администраторы.")
				continue
			}
			b.handleSubscribe(chatID)
		case cmd == "/unsubscribe" || text == "🔕 Отписаться":
			if chatID < 0 && !b.isGroupAdmin(chatID, fromID) {
				b.sendMessage(chatID, "⛔️ Отключать рассылку для группы могут только администраторы.")
				continue
			}
			b.handleUnsubscribe(chatID)
		case cmd == "/admin":
			b.handleAdmin(chatID, fromID, 0)
		case strings.HasPrefix(text, "/addadmin"):
			b.handleAddAdminCommand(chatID, fromID, text)
		case strings.HasPrefix(text, "/deladmin"):
			b.handleDelAdminCommand(chatID, fromID, text)
		default:
			// Для личных чатов выводим подсказку
			if chatID > 0 {
				b.sendMessage(chatID, "Используйте меню или команды:\n/today — Расписание на сегодня\n/city — Выбрать город или район РБ\n/settings — Настройки уведомлений\n/channel — Подключить Telegram-канал\n/subscribe — Подписаться на рассылку\n/unsubscribe — Отписаться\n/admin — Панель администратора")
			}
		}
	}
}

// handleGroupWelcome отправляет приветственное сообщение при добавлении бота в группу
func (b *Bot) handleGroupWelcome(chatID int64) {
	city, _ := b.storage.GetUserCity(chatID, b.defaultCity)
	welcomeText := fmt.Sprintf(
		"👋 *Ассаляму алейкум!*\n\n"+
			"Спасибо за добавление бота в группу!\n"+
			"Бот может ежедневно присылать расписание намаза (ДУМ РБ) и точное время восхода солнца (voshod-solnca.ru) для городов и районов Башкортостана.\n\n"+
			"📍 Текущий населенный пункт: *%s*\n\n"+
			"📌 *Команды для группы:*\n"+
			"• /today — Показать расписание на сегодня\n"+
			"• /settings — Настроить город и время рассылки для группы (в ЛС)\n"+
			"• /subscribe — Включить ежедневную рассылку в эту группу\n"+
			"• /unsubscribe — Отключить рассылку",
		city,
	)
	b.sendMessage(chatID, welcomeText)
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
	eveningTime := ""
	notify15min := true
	notifyAtTime := true

	if u != nil {
		if u.DailyScheduleTime != "" {
			selectedTime = u.DailyScheduleTime
		}
		eveningTime = u.EveningScheduleTime
		notify15min = u.Notify15Min
		notifyAtTime = u.NotifyAtTime
	}

	eveningDisplay := "Отключена"
	if eveningTime != "" {
		eveningDisplay = eveningTime
	}

	currentCity, _ := b.storage.GetUserCity(chatID, b.defaultCity)

	text := fmt.Sprintf(
		"⚙️ *Настройки рассылки и напоминаний*\n\n"+
			"📍 *Текущий город/район:* %s\n"+
			"🌅 *Утренняя рассылка (на день):* %s\n"+
			"🌙 *Вечерняя рассылка (на вечер и завтра):* %s\n\n"+
			"Нажмите на нужную кнопку для изменения параметров:",
		escapeMarkdown(currentCity), selectedTime, eveningDisplay,
	)

	// 1. Утреннее время
	times := []string{"05:00", "06:00", "07:00", "08:00", "09:00", "10:00"}
	var morningRow1 []tgbotapi.InlineKeyboardButton
	var morningRow2 []tgbotapi.InlineKeyboardButton

	for i, t := range times {
		label := t
		if t == selectedTime {
			label = "✓ " + t
		}
		btn := tgbotapi.NewInlineKeyboardButtonData(label, "set_time:"+t)
		if i < 3 {
			morningRow1 = append(morningRow1, btn)
		} else {
			morningRow2 = append(morningRow2, btn)
		}
	}

	// 2. Вечернее время
	etimes := []struct {
		val   string
		label string
	}{
		{"off", "🌙 Откл"},
		{"17:00", "17:00"},
		{"18:00", "18:00"},
		{"19:00", "19:00"},
		{"20:00", "20:00"},
		{"21:00", "21:00"},
	}
	var eveningRow1 []tgbotapi.InlineKeyboardButton
	var eveningRow2 []tgbotapi.InlineKeyboardButton

	for i, et := range etimes {
		display := et.label
		if (et.val == "off" && eveningTime == "") || (et.val == eveningTime) {
			display = "✓ " + et.label
		}
		btn := tgbotapi.NewInlineKeyboardButtonData(display, "set_etime:"+et.val)
		if i < 3 {
			eveningRow1 = append(eveningRow1, btn)
		} else {
			eveningRow2 = append(eveningRow2, btn)
		}
	}

	label15 := "⏳ 15 мин: [ ]"
	if notify15min {
		label15 = "⏳ 15 мин: [✓]"
	}

	labelAtTime := "🔔 В намаз: [ ]"
	if notifyAtTime {
		labelAtTime = "🔔 В намаз: [✓]"
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		morningRow1,
		morningRow2,
		eveningRow1,
		eveningRow2,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label15, "toggle_15min"),
			tgbotapi.NewInlineKeyboardButtonData(labelAtTime, "toggle_attime"),
		),
	)

	if messageID > 0 {
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
		editMsg.ParseMode = "Markdown"
		editMsg.ReplyMarkup = &keyboard
		if _, err := b.api.Send(editMsg); err != nil && !strings.Contains(err.Error(), "message is not modified") {
			log.Printf("Ошибка редактирования handleSettings: %v", err)
		}
	} else {
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard
		b.api.Send(msg)
	}
}

func (b *Bot) handleCallback(cb *tgbotapi.CallbackQuery) {
	if cb == nil {
		return
	}
	if b.handleAdminCallbacks(cb) {
		return
	}
	if b.handleGroupCallbacks(cb) {
		return
	}

	fromID := int64(0)
	if cb.From != nil {
		fromID = cb.From.ID
	}

	if cb.Message == nil {
		return
	}

	chatID := cb.Message.Chat.ID
	messageID := cb.Message.MessageID
	data := cb.Data

	// Защита: если callback вызван внутри группы, проверять права админа группы
	if chatID < 0 && !b.isGroupAdmin(chatID, fromID) {
		b.api.Send(tgbotapi.NewCallbackWithAlert(cb.ID, "⛔️ Настройки группы могут изменять только администраторы!"))
		return
	}

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
		b.answerCallback(cb.ID, "Утренняя рассылка: "+timeStr)
		b.handleSettings(chatID, messageID)
		return
	}

	if strings.HasPrefix(data, "set_etime:") {
		timeStr := strings.TrimPrefix(data, "set_etime:")
		if timeStr == "off" {
			_ = b.storage.UpdateEveningTime(chatID, "")
			b.answerCallback(cb.ID, "Вечерняя рассылка отключена")
		} else {
			_ = b.storage.UpdateEveningTime(chatID, timeStr)
			b.answerCallback(cb.ID, "Вечерняя рассылка: "+timeStr)
		}
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

func (b *Bot) handleStart(chatID int64, fromID int64, parts []string) {
	// Проверка deep link аргумента (например: /start grp_-100123456789)
	if len(parts) > 1 {
		arg := parts[1]
		var targetGroupID int64
		if strings.HasPrefix(arg, "grp_") {
			_, _ = fmt.Sscanf(strings.TrimPrefix(arg, "grp_"), "%d", &targetGroupID)
		} else if strings.HasPrefix(arg, "group_") {
			_, _ = fmt.Sscanf(strings.TrimPrefix(arg, "group_"), "%d", &targetGroupID)
		}

		if targetGroupID != 0 {
			if targetGroupID > 0 {
				targetGroupID = -targetGroupID
			}
			if !b.isGroupAdmin(targetGroupID, fromID) {
				b.sendMessage(chatID, "⛔️ *Доступ ограничен*\n\nВы не являетесь администратором данной группы или бот не добавлен в неё.")
				return
			}
			groupTitle := b.getGroupTitle(targetGroupID)
			b.handleGroupSettings(chatID, targetGroupID, groupTitle, 0)
			return
		}
	}

	city, _ := b.storage.GetUserCity(chatID, b.defaultCity)
	msgText := fmt.Sprintf(
		"Ассаляму алейкум! 🖐\n\n"+
			"Этот бот показывает расписание намаза (ДУМ РБ) и точное время восхода солнца (voshod-solnca.ru) для населенных пунктов Республики Башкортостан.\n\n"+
			"📍 Ваш текущий выбор: *%s*\n\n"+
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

