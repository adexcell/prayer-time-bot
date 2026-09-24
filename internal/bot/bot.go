package bot

import (
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"namaz-time-bot/internal/api"
	"namaz-time-bot/internal/storage"
)

type pendingActionType string

const (
	pendingActionNone          pendingActionType = ""
	pendingActionHeader        pendingActionType = "header"
	pendingActionFooter        pendingActionType = "footer"
	pendingActionPrayer        pendingActionType = "prayer"
	pendingActionBroadcastTime pendingActionType = "broadcast_time"
	pendingActionFixedTime     pendingActionType = "fixed_time"
)

type userState struct {
	action        pendingActionType
	targetGroupID int64
	prayerKey     string
	cityID        int
}

type Bot struct {
	api             *tgbotapi.BotAPI
	client          *api.Client
	storage         *storage.Storage
	defaultCity     string
	configAdminIDs  []int64
	userStates      map[int64]userState
	stateMu         sync.RWMutex
	previewMessages map[int64]int
	previewMu       sync.Mutex
}

func New(token, defaultCity string, configAdminIDs []int64, client *api.Client, store *storage.Storage) (*Bot, error) {
	botAPI, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	log.Printf("Авторизован аккаунт бота: %s", botAPI.Self.UserName)

	return &Bot{
		api:             botAPI,
		client:          client,
		storage:         store,
		defaultCity:     defaultCity,
		configAdminIDs:  configAdminIDs,
		userStates:      make(map[int64]userState),
		previewMessages: make(map[int64]int),
	}, nil
}

func (b *Bot) setUserState(userID int64, state userState) {
	b.stateMu.Lock()
	defer b.stateMu.Unlock()
	b.userStates[userID] = state
}

func (b *Bot) getUserState(userID int64) (userState, bool) {
	b.stateMu.RLock()
	defer b.stateMu.RUnlock()
	state, exists := b.userStates[userID]
	return state, exists
}

func (b *Bot) clearUserState(userID int64) {
	b.stateMu.Lock()
	defer b.stateMu.Unlock()
	delete(b.userStates, userID)
}

func (b *Bot) setPreviewMessage(userID int64, msgID int) {
	b.previewMu.Lock()
	defer b.previewMu.Unlock()
	b.previewMessages[userID] = msgID
}

func (b *Bot) getAndClearPreviewMessage(userID int64) int {
	b.previewMu.Lock()
	defer b.previewMu.Unlock()
	msgID := b.previewMessages[userID]
	delete(b.previewMessages, userID)
	return msgID
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

		// Обработка ввода текста для настройки оформления (заголовок, подпись, названия молитв, время рассылки) в ЛС
		if chatID > 0 {
			if state, ok := b.getUserState(fromID); ok && state.action != pendingActionNone {
				b.clearUserState(fromID)
				if text == "-" || text == "сброс" || text == "/cancel" || strings.ToLower(text) == "отмена" {
					if state.action == pendingActionHeader {
						_ = b.storage.UpdateChatHeader(state.targetGroupID, "")
						b.sendMessage(chatID, "✅ Заголовок сброшен по умолчанию.")
						b.handleGroupStyleSettings(chatID, state.targetGroupID, 0)
					} else if state.action == pendingActionFooter {
						_ = b.storage.UpdateChatFooter(state.targetGroupID, "")
						b.sendMessage(chatID, "✅ Подпись (footer) удалена.")
						b.handleGroupStyleSettings(chatID, state.targetGroupID, 0)
					} else if state.action == pendingActionPrayer {
						_ = b.storage.UpdateCustomPrayerName(state.targetGroupID, state.prayerKey, "")
						b.sendMessage(chatID, "✅ Название молитвы сброшено к выбранному стилю.")
						b.handleGroupPrayersMenu(chatID, state.targetGroupID, 0)
					} else if state.action == pendingActionBroadcastTime {
						b.sendMessage(chatID, "❌ Ввод времени рассылки отменен.")
						if state.targetGroupID < 0 {
							b.handleGroupSettings(chatID, state.targetGroupID, b.getGroupTitle(state.targetGroupID), 0)
						} else {
							b.handleSettings(chatID, 0)
						}
					} else if state.action == pendingActionFixedTime {
						b.sendMessage(chatID, "❌ Ввод фиксированного времени отменен.")
						b.handleAdminSelectOffsetStep(chatID, state.cityID, state.prayerKey, 0)
					}
				} else {
					if state.action == pendingActionHeader {
						_ = b.storage.UpdateChatHeader(state.targetGroupID, text)
						b.sendMessage(chatID, "✅ Заголовок рассылки успешно сохранен!")
						b.handleGroupStyleSettings(chatID, state.targetGroupID, 0)
					} else if state.action == pendingActionFooter {
						_ = b.storage.UpdateChatFooter(state.targetGroupID, text)
						b.sendMessage(chatID, "✅ Подпись (footer) рассылки успешно сохранена!")
						b.handleGroupStyleSettings(chatID, state.targetGroupID, 0)
					} else if state.action == pendingActionPrayer {
						_ = b.storage.UpdateCustomPrayerName(state.targetGroupID, state.prayerKey, text)
						b.sendMessage(chatID, "✅ Название молитвы успешно сохранено!")
						b.handleGroupPrayersMenu(chatID, state.targetGroupID, 0)
					} else if state.action == pendingActionBroadcastTime {
						normTime, err := b.storage.AddBroadcastTime(state.targetGroupID, text)
						if err != nil {
							// Возвращаем состояние, чтобы дать пользователю ввести время снова
							b.setUserState(fromID, state)
							b.sendMessage(chatID, "❌ Некорректный формат времени. Введите время в формате `ЧЧ:ММ` (например: `06:30` или `19:00`):")
							continue
						}
						b.sendMessage(chatID, fmt.Sprintf("✅ Время рассылки *%s* успешно добавлено!", normTime))
						if state.targetGroupID < 0 {
							b.handleGroupSettings(chatID, state.targetGroupID, b.getGroupTitle(state.targetGroupID), 0)
						} else {
							b.handleSettings(chatID, 0)
						}
					} else if state.action == pendingActionFixedTime {
						timeStr := strings.TrimSpace(text)
						parts := strings.Split(timeStr, ":")
						var hour, min int
						valid := false
						if len(parts) == 2 {
							if h, errH := strconv.Atoi(parts[0]); errH == nil && h >= 0 && h <= 23 {
								if m, errM := strconv.Atoi(parts[1]); errM == nil && m >= 0 && m <= 59 {
									hour = h
									min = m
									valid = true
								}
							}
						}
						if !valid {
							b.setUserState(fromID, state)
							b.sendMessage(chatID, "❌ Некорректный формат времени. Введите время в формате `ЧЧ:ММ` (например: `13:30` или `06:00`):")
							continue
						}
						fixedTime := fmt.Sprintf("%02d:%02d", hour, min)
						b.handleAdminSelectDurationStepForFixed(chatID, state.cityID, state.prayerKey, fixedTime, 0)
					}
				}
				continue
			}
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
		case cmd == "/channels" || cmd == "/mychannels":
			b.handleMyChannelsList(chatID, fromID, 1, 0)
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
				b.sendMessage(chatID, "Используйте меню или команды:\n/today — Расписание на сегодня\n/city — Выбрать город или район РБ\n/settings — Настройки уведомлений\n/channels — Мои каналы и группы\n/channel — Подключить Telegram-канал\n/subscribe — Подписаться на рассылку\n/unsubscribe — Отписаться\n/admin — Панель администратора")
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

	notify15min := true
	notifyAtTime := true
	var broadcastTimes []string

	if u != nil {
		broadcastTimes = u.GetParsedBroadcastTimes()
		notify15min = u.Notify15Min
		notifyAtTime = u.NotifyAtTime
	}

	timesDisplay := "🔕 Отключена"
	if len(broadcastTimes) > 0 {
		timesDisplay = strings.Join(broadcastTimes, ", ")
	}

	currentCity, _ := b.storage.GetUserCity(chatID, b.defaultCity)

	text := fmt.Sprintf(
		"⚙️ *Настройки рассылки и напоминаний*\n\n"+
			"📍 *Текущий город/район:* %s\n"+
			"⏰ *Времена рассылки:* %s\n\n"+
			"Нажмите *«➕ Добавить время»*, чтобы ввести время (ЧЧ:ММ), или нажмите на крестик у времени для его удаления:",
		escapeMarkdown(currentCity), escapeMarkdown(timesDisplay),
	)

	var keyboardRows [][]tgbotapi.InlineKeyboardButton

	// 1. Кнопки удаления имеющихся времен рассылки (по 3 в строке)
	if len(broadcastTimes) > 0 {
		var curRow []tgbotapi.InlineKeyboardButton
		for _, t := range broadcastTimes {
			btn := tgbotapi.NewInlineKeyboardButtonData("✕ "+t, "user_del_time:"+t)
			curRow = append(curRow, btn)
			if len(curRow) == 3 {
				keyboardRows = append(keyboardRows, curRow)
				curRow = []tgbotapi.InlineKeyboardButton{}
			}
		}
		if len(curRow) > 0 {
			keyboardRows = append(keyboardRows, curRow)
		}

		// Строка добавления и очистки
		keyboardRows = append(keyboardRows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить время", "user_add_time"),
			tgbotapi.NewInlineKeyboardButtonData("🗑 Очистить все", "user_clear_times"),
		})
	} else {
		// Если времен нет
		keyboardRows = append(keyboardRows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить время рассылки", "user_add_time"),
		})
	}

	label15 := "⏳ 15 мин: [ ]"
	if notify15min {
		label15 = "⏳ 15 мин: [✓]"
	}

	labelAtTime := "🔔 В намаз: [ ]"
	if notifyAtTime {
		labelAtTime = "🔔 В намаз: [✓]"
	}

	keyboardRows = append(keyboardRows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(label15, "toggle_15min"),
		tgbotapi.NewInlineKeyboardButtonData(labelAtTime, "toggle_attime"),
	})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)

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

	if data == "user_add_time" {
		b.setUserState(fromID, userState{
			action:        pendingActionBroadcastTime,
			targetGroupID: chatID,
		})
		prompt := "⏰ *Введите время для рассылки в формате ЧЧ:ММ*\n\nНапример: `07:00` или `19:30`.\n\nОтправьте время ответным сообщением (или `/cancel` для отмены)."
		b.sendMessage(chatID, prompt)
		b.answerCallback(cb.ID, "")
		return
	}

	if strings.HasPrefix(data, "user_del_time:") {
		timeStr := strings.TrimPrefix(data, "user_del_time:")
		_ = b.storage.RemoveBroadcastTime(chatID, timeStr)
		b.answerCallback(cb.ID, "Время "+timeStr+" удалено")
		b.handleSettings(chatID, messageID)
		return
	}

	if data == "user_clear_times" {
		_ = b.storage.ClearBroadcastTimes(chatID)
		b.answerCallback(cb.ID, "Все времена рассылки очищены")
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
	b.sendMessageWithShare(chatID, msgText)
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

// CreateShareInlineKeyboard создает кнопки «Поделиться» для WhatsApp и MAX
func CreateShareInlineKeyboard(text string) tgbotapi.InlineKeyboardMarkup {
	encodedText := url.QueryEscape(text)
	waURL := "https://api.whatsapp.com/send?text=" + encodedText
	maxURL := "https://max.ru/share?text=" + encodedText

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("WhatsApp ↗", waURL),
			tgbotapi.NewInlineKeyboardButtonURL("MAX ↗", maxURL),
		),
	)
}

func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}

func (b *Bot) sendMessageWithShare(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = CreateShareInlineKeyboard(text)
	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения с кнопками 'Поделиться': %v", err)
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

// SendToChat используется планировщиком для рассылки обычных сообщений
func (b *Bot) SendToChat(chatID int64, text string) {
	b.sendMessage(chatID, text)
}

// SendToChatWithShare используется планировщиком для рассылки расписания с кнопками «Поделиться»
func (b *Bot) SendToChatWithShare(chatID int64, text string) {
	b.sendMessageWithShare(chatID, text)
}

