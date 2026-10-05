package bot

import (
	"fmt"
	"log"
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
	pendingActionNone           pendingActionType = ""
	pendingActionHeader         pendingActionType = "header"
	pendingActionFooter         pendingActionType = "footer"
	pendingActionPrayer         pendingActionType = "prayer"
	pendingActionBroadcastTime  pendingActionType = "broadcast_time"
	pendingActionFixedTime      pendingActionType = "fixed_time"
	pendingActionHijriDay       pendingActionType = "hijri_day"
	pendingActionHadithText     pendingActionType = "hadith_text"
	pendingActionHadithSource   pendingActionType = "hadith_source"
	pendingActionHadithTime     pendingActionType = "hadith_time"
	pendingActionWelcomeMessage pendingActionType = "welcome_message"
)

const DefaultWelcomeMessageTemplate = "Ассаляму алейкум! 🖐\n\n" +
	"Этот бот показывает расписание намаза (ДУМ РБ) и точное время восхода солнца (voshod-solnca.ru) для населенных пунктов Республики Башкортостан.\n\n" +
	"📍 Ваш текущий выбор: *{city}*\n\n" +
	"Выберите действие в меню ниже:"

type userState struct {
	action        pendingActionType
	targetGroupID int64
	prayerKey     string
	cityID        int
	hadithCat     string
	hadithCol     string
	hadithText    string
	settingKey    string
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

	// Регистрируем системное меню команд Telegram (кнопка Menu/[/] возле поля ввода)
	commandsConfig := tgbotapi.NewSetMyCommands(
		tgbotapi.BotCommand{Command: "start", Description: "Главное меню бота"},
		tgbotapi.BotCommand{Command: "today", Description: "Расписание на сегодня"},
		tgbotapi.BotCommand{Command: "digest", Description: "Дайджест выбранных городов"},
		tgbotapi.BotCommand{Command: "city", Description: "Выбрать город или район"},
		tgbotapi.BotCommand{Command: "settings", Description: "Настройки рассылки и оформления"},
		tgbotapi.BotCommand{Command: "hadith", Description: "Хадис дня (Бухари, Муслим, Сады праведных)"},
		tgbotapi.BotCommand{Command: "channels", Description: "Мои подконтрольные каналы и группы"},
		tgbotapi.BotCommand{Command: "subscribe", Description: "Включить ежедневную рассылку"},
		tgbotapi.BotCommand{Command: "unsubscribe", Description: "Отключить рассылку"},
		tgbotapi.BotCommand{Command: "admin", Description: "Панель администратора"},
	)
	if _, err := botAPI.Request(commandsConfig); err != nil {
		log.Printf("Предупреждение: Не удалось зарегистрировать меню команд Telegram: %v", err)
	}

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
					} else if state.action == pendingActionHijriDay {
						b.sendMessage(chatID, "❌ Установка числа месяца отменена.")
						b.handleAdminHijri(chatID, fromID, 0)
					} else if state.action == pendingActionHadithText || state.action == pendingActionHadithSource {
						b.sendMessage(chatID, "❌ Добавление хадиса отменено.")
						b.handleAdminHadithsMenu(chatID, fromID, 0)
					} else if state.action == pendingActionHadithTime {
						b.sendMessage(chatID, "❌ Изменение времени рассылки отменено.")
						b.handleAdminHadithsConfig(chatID, fromID, 0)
					} else if state.action == pendingActionWelcomeMessage {
						b.sendMessage(chatID, "❌ Изменение приветственного сообщения отменено.")
						b.handleAdminWelcome(chatID, fromID, 0)
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
					} else if state.action == pendingActionHijriDay {
						if !b.storage.IsAdmin(fromID, b.configAdminIDs) {
							b.sendMessage(chatID, "⛔️ Корректировка календаря доступна только администраторам.")
							continue
						}
						targetDay, err := strconv.Atoi(strings.TrimSpace(text))
						if err != nil || targetDay < 1 || targetDay > 30 {
							b.setUserState(fromID, state)
							b.sendMessage(chatID, "❌ Некорректное число дня. Введите число месяца от 1 до 30 (например: `2`):")
							continue
						}
						now := time.Now()
						offset, errFind := api.FindOffsetForTargetDay(now, targetDay)
						if errFind != nil {
							b.setUserState(fromID, state)
							b.sendMessage(chatID, fmt.Sprintf("❌ %v. Попробуйте еще раз или используйте кнопки смещения в панели.", errFind))
							continue
						}
						_ = b.storage.SetHijriOffset(offset)
						api.SetGlobalHijriOffset(offset)

						adjHD, _ := api.GetHijriDate(now, offset)
						monthName := api.GetHijriMonthName(adjHD.Month, "ru")
						b.sendMessage(chatID, fmt.Sprintf("✅ Календарь успешно скорректирован!\n\nСегодняшний день установлен как: *%d %s %d г. х.*\nСмещение: *%+d дн.*\n\nВесь календарь автоматически сдвинут.", adjHD.Day, monthName, adjHD.Year, offset))
						b.handleAdminHijri(chatID, fromID, 0)
					} else if state.action == pendingActionHadithText {
						b.setUserState(fromID, userState{
							action:     pendingActionHadithSource,
							hadithCat:  state.hadithCat,
							hadithCol:  state.hadithCol,
							hadithText: text,
						})
						prompt := "📝 *Текст хадиса принят!*\n\n" +
							"Теперь введите источник хадиса (например: `Сахих аль-Бухари, 1958` или `Сады праведных, 1258`):\n\n" +
							"Отправьте `-` для отмены."
						b.sendMessage(chatID, prompt)
					} else if state.action == pendingActionHadithSource {
						hID, errAdd := b.storage.AddHadith(state.hadithText, text, state.hadithCol, state.hadithCat)
						if errAdd != nil {
							b.sendMessage(chatID, fmt.Sprintf("❌ Ошибка сохранения хадиса: %v", errAdd))
						} else {
							b.sendMessage(chatID, fmt.Sprintf(
								"✅ *Хадис #%d успешно сохранен!*\n\n"+
									"Категория: *%s*\n"+
									"Сборник: *%s*\n"+
									"Источник: *%s*",
								hID,
								storage.GetCategoryTitle(state.hadithCat),
								storage.GetCollectionTitle(state.hadithCol),
								escapeMarkdown(text),
							))
						}
						b.handleAdminHadithsMenu(chatID, fromID, 0)
					} else if state.action == pendingActionHadithTime {
						normTime, errNorm := storage.NormalizeBroadcastTime(text)
						if errNorm != nil {
							b.setUserState(fromID, state)
							b.sendMessage(chatID, "❌ Некорректный формат времени. Введите время в формате `ЧЧ:ММ` (например: `09:00`):")
							continue
						}
						_ = b.storage.SetSetting(state.settingKey, normTime)
						b.sendMessage(chatID, fmt.Sprintf("✅ Время рассылки успешно установлено: *%s*", normTime))
						b.handleAdminHadithsConfig(chatID, fromID, 0)
					} else if state.action == pendingActionWelcomeMessage {
						_ = b.storage.SetWelcomeMessage(text)
						b.sendMessage(chatID, "✅ Приветственное сообщение успешно обновлено!")
						b.handleAdminWelcome(chatID, fromID, 0)
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
		case cmd == "/today" || text == "🕌 Расписание на сегодня" || text == "🕌 Сегодня":
			b.handleToday(chatID)
		case cmd == "/digest" || text == "📋 Дайджест (WA/MAX)" || text == "📋 Дайджест рассылки" || text == "📋 Дайджест":
			b.handleDigest(chatID, 0)
		case cmd == "/city" || text == "🏙 Выбрать город" || text == "🏙 Город":
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
		case cmd == "/channels" || cmd == "/mychannels" || text == "📢 Мои каналы":
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
		case cmd == "/hadith" || text == "📖 Хадис дня" || text == "📖 Хадис":
			b.handleRandomHadith(chatID)
		case cmd == "/admin":
			b.handleAdmin(chatID, fromID, 0)
		case strings.HasPrefix(text, "/addadmin"):
			b.handleAddAdminCommand(chatID, fromID, text)
		case strings.HasPrefix(text, "/deladmin"):
			b.handleDelAdminCommand(chatID, fromID, text)
		case strings.HasPrefix(text, "/hijri"):
			b.handleHijriCommand(chatID, fromID, text)
		default:
			// Для личных чатов выводим подсказку
			if chatID > 0 {
				b.sendMessage(chatID, "Используйте меню или команды:\n/start — Главное меню\n/today — Расписание на сегодня\n/digest — Дайджест городов\n/city — Выбрать город или район РБ\n/hadith — Случайный хадис дня\n/settings — Настройки уведомлений\n/channels — Мои каналы и группы\n/channel — Подключить Telegram-канал\n/subscribe — Подписаться на рассылку\n/unsubscribe — Отписаться\n/admin — Панель администратора")
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
		editMsg.DisableWebPagePreview = true
		editMsg.ReplyMarkup = &keyboard
		if _, err := b.api.Send(editMsg); err != nil {
			if !strings.Contains(err.Error(), "message is not modified") {
				log.Printf("Ошибка редактирования сообщения в handleChooseLocation: %v", err)
			}
		}
	} else {
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		msg.DisableWebPagePreview = true
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
	hadithDaily := true
	hadithPrayer := true
	var broadcastTimes []string

	if u != nil {
		broadcastTimes = u.GetParsedBroadcastTimes()
		notify15min = u.Notify15Min
		notifyAtTime = u.NotifyAtTime
		hadithDaily = u.HadithDailyEnabled
		hadithPrayer = u.HadithPrayerEnabled
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

	labelHadithDaily := "📖 Хадис дня: [ ]"
	if hadithDaily {
		labelHadithDaily = "📖 Хадис дня: [✓]"
	}

	labelHadithPrayer := "📖 Хадисы к намазам: [ ]"
	if hadithPrayer {
		labelHadithPrayer = "📖 Хадисы к намазам: [✓]"
	}

	keyboardRows = append(keyboardRows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(label15, "toggle_15min"),
		tgbotapi.NewInlineKeyboardButtonData(labelAtTime, "toggle_attime"),
	})

	keyboardRows = append(keyboardRows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(labelHadithDaily, "toggle_hadith_daily"),
		tgbotapi.NewInlineKeyboardButtonData(labelHadithPrayer, "toggle_hadith_prayer"),
	})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)

	if b.api == nil {
		return
	}

	if messageID > 0 {
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
		editMsg.ParseMode = "Markdown"
		editMsg.DisableWebPagePreview = true
		editMsg.ReplyMarkup = &keyboard
		if _, err := b.api.Send(editMsg); err != nil && !strings.Contains(err.Error(), "message is not modified") {
			log.Printf("Ошибка редактирования handleSettings: %v", err)
		}
	} else {
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		msg.DisableWebPagePreview = true
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

	if strings.HasPrefix(data, "fav_add_select") {
		b.handleChooseFavLocation(chatID, true, 1, messageID)
		b.answerCallback(cb.ID, "")
		return
	}

	if strings.HasPrefix(data, "fav_tab_cities:") {
		pageStr := strings.TrimPrefix(data, "fav_tab_cities:")
		var page int
		fmt.Sscanf(pageStr, "%d", &page)
		b.handleChooseFavLocation(chatID, true, page, messageID)
		b.answerCallback(cb.ID, "")
		return
	}

	if strings.HasPrefix(data, "fav_tab_districts:") {
		pageStr := strings.TrimPrefix(data, "fav_tab_districts:")
		var page int
		fmt.Sscanf(pageStr, "%d", &page)
		b.handleChooseFavLocation(chatID, false, page, messageID)
		b.answerCallback(cb.ID, "")
		return
	}

	if strings.HasPrefix(data, "fav_add_city:") {
		rest := strings.TrimPrefix(data, "fav_add_city:")
		parts := strings.Split(rest, ":")
		var cityID int
		isCityTab := true
		page := 1

		if len(parts) == 3 {
			if parts[0] == "d" {
				isCityTab = false
			}
			_, _ = fmt.Sscanf(parts[1], "%d", &page)
			_, _ = fmt.Sscanf(parts[2], "%d", &cityID)
		} else if len(parts) == 1 {
			_, _ = fmt.Sscanf(parts[0], "%d", &cityID)
		}

		cityName := ""
		if cityID > 0 {
			if cityInfo, ok := api.GetCityByID(cityID); ok {
				cityName = cityInfo.DisplayName
			}
		}
		if cityName == "" {
			cityName = rest
		}

		favCities, _ := b.storage.GetFavoriteCities(chatID)
		isFav := false
		for _, c := range favCities {
			if c == cityName {
				isFav = true
				break
			}
		}

		if isFav {
			_ = b.storage.RemoveFavoriteCity(chatID, cityName)
			b.answerCallback(cb.ID, "Удалено из дайджеста: "+cityName)
		} else {
			_ = b.storage.AddFavoriteCity(chatID, cityName)
			b.answerCallback(cb.ID, "Добавлено в дайджест: "+cityName)
		}
		b.handleChooseFavLocation(chatID, isCityTab, page, messageID)
		return
	}

	if data == "fav_back_manage" {
		b.handleDigestManagePanel(chatID, messageID)
		b.answerCallback(cb.ID, "")
		return
	}

	if strings.HasPrefix(data, "fav_del:") {
		cityName := strings.TrimPrefix(data, "fav_del:")
		_ = b.storage.RemoveFavoriteCity(chatID, cityName)
		b.answerCallback(cb.ID, "Удалено из дайджеста: "+cityName)
		b.handleDigestManagePanel(chatID, messageID)
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
	case "toggle_hadith_daily":
		_ = b.storage.ToggleHadithDaily(chatID)
		b.answerCallback(cb.ID, "Настройка рассылки хадиса дня изменена")
		b.handleSettings(chatID, messageID)
	case "toggle_hadith_prayer":
		_ = b.storage.ToggleHadithPrayer(chatID)
		b.answerCallback(cb.ID, "Настройка хадисов к намазам изменена")
		b.handleSettings(chatID, messageID)
	}
}

func (b *Bot) answerCallback(callbackID, text string) {
	if b.api == nil {
		return
	}
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
	msgText := b.getWelcomeMessage(city)
	b.sendMessageWithKeyboard(chatID, msgText)
}

func (b *Bot) getWelcomeMessage(city string) string {
	custom := strings.TrimSpace(b.storage.GetWelcomeMessage())
	if custom != "" {
		text := strings.ReplaceAll(custom, "{city}", city)
		text = strings.ReplaceAll(text, "{город}", city)
		if !strings.Contains(custom, "{city}") && !strings.Contains(custom, "{город}") {
			text += fmt.Sprintf("\n\n📍 Ваш текущий выбор: *%s*\n\nВыберите действие в меню ниже:", city)
		}
		return text
	}
	return strings.ReplaceAll(DefaultWelcomeMessageTemplate, "{city}", city)
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

func (b *Bot) sendMessage(chatID int64, text string) {
	if b.api == nil {
		return
	}
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = true
	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}

func (b *Bot) sendMessageWithShare(chatID int64, text string) {
	b.sendMessage(chatID, text)
}

func (b *Bot) sendMessageWithKeyboard(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = true

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🕌 Сегодня"),
			tgbotapi.NewKeyboardButton("📋 Дайджест"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🏙 Город"),
			tgbotapi.NewKeyboardButton("⚙️ Настройки"),
			tgbotapi.NewKeyboardButton("📖 Хадис"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📢 Мои каналы"),
		),
	)
	keyboard.ResizeKeyboard = true
	msg.ReplyMarkup = keyboard

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения с клавиатурой: %v", err)
	}
}

// handleRandomHadith отправляет пользователю случайный достоверный хадис
func (b *Bot) handleRandomHadith(chatID int64) {
	h, err := b.storage.GetRandomHadithAny()
	if err != nil || h == nil {
		b.sendMessage(chatID, "📖 В данный момент хадисы недоступны.")
		return
	}
	msg := fmt.Sprintf("📖 *Хадис дня*\n\n«%s»\n\n📚 *Источник:* %s", h.Text, h.Source)
	b.sendMessageWithShare(chatID, msg)
}

// handleDigest отправляет расписание для всех выбранных в дайджест городов
func (b *Bot) handleDigest(chatID int64, messageID int) {
	cities, err := b.storage.GetFavoriteCities(chatID)
	if err != nil || len(cities) == 0 {
		defaultCity, _ := b.storage.GetUserCity(chatID, b.defaultCity)
		cities = []string{defaultCity}
	}

	now := time.Now()
	headerText := fmt.Sprintf("📋 *Дайджест расписания намазов (%s)*\n\n"+
		"Ниже выведены карточки расписания для ваших городов:", now.Format("02.01.2006"))
	b.sendMessage(chatID, headerText)

	for _, city := range cities {
		item, err := b.client.FetchPrayerTimes(city, now)
		if err != nil {
			log.Printf("Ошибка получения времени намаза для %s в дайджесте: %v", city, err)
			continue
		}
		msgText := api.FormatMessage(item, city, now)
		b.sendMessageWithShare(chatID, msgText)
	}

	b.handleDigestManagePanel(chatID, 0)
}

// handleDigestManagePanel отображает и обновляет только панель управления городами дайджеста
func (b *Bot) handleDigestManagePanel(chatID int64, messageID int) {
	cities, err := b.storage.GetFavoriteCities(chatID)
	if err != nil || len(cities) == 0 {
		defaultCity, _ := b.storage.GetUserCity(chatID, b.defaultCity)
		cities = []string{defaultCity}
	}

	citiesList := strings.Join(cities, ", ")
	manageText := fmt.Sprintf("⚙️ *Ваши города для дайджеста:* %s\n\n"+
		"Вы можете добавить новые города или удалить ненужные кнопками ниже:", escapeMarkdown(citiesList))

	var rows [][]tgbotapi.InlineKeyboardButton
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("➕ Добавить город в дайджест", "fav_add_select:1"),
	})

	if len(cities) > 0 {
		var delRow []tgbotapi.InlineKeyboardButton
		for _, c := range cities {
			btn := tgbotapi.NewInlineKeyboardButtonData("✕ "+c, "fav_del:"+c)
			delRow = append(delRow, btn)
			if len(delRow) == 2 {
				rows = append(rows, delRow)
				delRow = []tgbotapi.InlineKeyboardButton{}
			}
		}
		if len(delRow) > 0 {
			rows = append(rows, delRow)
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	if messageID > 0 {
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, manageText)
		editMsg.ParseMode = "Markdown"
		editMsg.DisableWebPagePreview = true
		editMsg.ReplyMarkup = &keyboard
		if _, err := b.api.Send(editMsg); err != nil && !strings.Contains(err.Error(), "message is not modified") {
			log.Printf("Ошибка редактирования панели дайджеста: %v", err)
		}
	} else {
		msg := tgbotapi.NewMessage(chatID, manageText)
		msg.ParseMode = "Markdown"
		msg.DisableWebPagePreview = true
		msg.ReplyMarkup = keyboard
		b.api.Send(msg)
	}
}

// handleChooseFavLocation позволяет выбрать город для добавления в дайджест
func (b *Bot) handleChooseFavLocation(chatID int64, isCityTab bool, page int, messageID int) {
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

	favCities, _ := b.storage.GetFavoriteCities(chatID)
	favMap := make(map[string]bool)
	for _, c := range favCities {
		favMap[c] = true
	}

	text := fmt.Sprintf(
		"➕ *Добавление города в ваш дайджест рассылки:*\n\n"+
			"Выберите населенный пункт из списка ниже:\n"+
			"📄 Страница: %d из %d",
		page, totalPages,
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
		tgbotapi.NewInlineKeyboardButtonData(citiesTabLabel, "fav_tab_cities:1"),
		tgbotapi.NewInlineKeyboardButtonData(districtsTabLabel, "fav_tab_districts:1"),
	}
	rows = append(rows, tabRow)

	// 2. Кнопки городов
	pageItems := items[startIdx:endIdx]
	var currentRow []tgbotapi.InlineKeyboardButton

	tabCode := "c"
	if !isCityTab {
		tabCode = "d"
	}

	for _, item := range pageItems {
		btnText := item.DisplayName
		if favMap[item.DisplayName] || favMap[item.CleanName] {
			btnText = "✓ " + item.DisplayName
		}
		btn := tgbotapi.NewInlineKeyboardButtonData(btnText, fmt.Sprintf("fav_add_city:%s:%d:%d", tabCode, page, item.ID))
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
	pagePrefix := "fav_tab_cities"
	if !isCityTab {
		pagePrefix = "fav_tab_districts"
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

	// 4. Кнопка возврата к управлению дайджестом
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к дайджесту", "fav_back_manage"),
	})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	if b.api == nil {
		return
	}

	if messageID > 0 {
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
		editMsg.ParseMode = "Markdown"
		editMsg.DisableWebPagePreview = true
		editMsg.ReplyMarkup = &keyboard
		_, _ = b.api.Send(editMsg)
	} else {
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		msg.DisableWebPagePreview = true
		msg.ReplyMarkup = keyboard
		_, _ = b.api.Send(msg)
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

