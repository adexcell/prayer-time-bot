package bot

import (
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"namaz-time-bot/internal/api"
)

// handleMyChatMember обрабатывает события добавления/изменения прав бота в каналах и группах
func (b *Bot) handleMyChatMember(m *tgbotapi.ChatMemberUpdated) {
	if m == nil {
		return
	}

	newStatus := m.NewChatMember.Status
	chatID := m.Chat.ID
	chatTitle := m.Chat.Title
	adminID := int64(0)
	if m.From.ID != 0 {
		adminID = m.From.ID
	}

	// Бот назначен администратором (в канале или группе)
	if newStatus == "administrator" {
		log.Printf("Бот назначен администратором в '%s' (ID: %d, Type: %s) пользователем ID: %d", chatTitle, chatID, m.Chat.Type, adminID)

		// Сохраняем информацию о чате и владельце
		_ = b.storage.SaveChatInfo(chatID, chatTitle, m.Chat.Type, adminID)

		// Автоматически подписываем чат на рассылку
		currentCity, _ := b.storage.GetUserCity(chatID, b.defaultCity)
		_ = b.storage.Subscribe(chatID, currentCity)

		chatTypeLabel := "группу"
		if m.Chat.IsChannel() {
			chatTypeLabel = "канал"
		}

		text := fmt.Sprintf(
			"🎉 *Бот успешно назначен администратором в %s «%s»!*\n\n"+
				"Теперь бот может автоматически публиковать ежедневное расписание намаза и напоминания.\n\n"+
				"👇 Нажмите кнопку ниже, чтобы настроить город, время рассылки и стиль сообщений:",
			chatTypeLabel, escapeMarkdown(chatTitle),
		)

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⚙️ Настроить расписание и стиль", fmt.Sprintf("gback:%d", chatID)),
			),
		)

		msg := tgbotapi.NewMessage(adminID, text)
		msg.ParseMode = "Markdown"
		msg.DisableWebPagePreview = true
		msg.ReplyMarkup = keyboard
		if _, err := b.api.Send(msg); err != nil {
			log.Printf("Не удалось отправить уведомление админу %d в ЛС: %v", adminID, err)
		}
	}
}

const channelsPageSize = 8

// handleMyChannelsList отображает список каналов и групп, доступных пользователю
func (b *Bot) handleMyChannelsList(userChatID int64, fromID int64, page int, messageID int) {
	isSuper := b.storage.IsAdmin(fromID, b.configAdminIDs)
	chats, err := b.storage.GetManagedChats(fromID, isSuper)
	if err != nil {
		log.Printf("Ошибка получения управляемых чатов: %v", err)
	}

	if page < 1 {
		page = 1
	}

	totalItems := len(chats)
	totalPages := (totalItems + channelsPageSize - 1) / channelsPageSize
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	startIdx := (page - 1) * channelsPageSize
	endIdx := startIdx + channelsPageSize
	if endIdx > totalItems {
		endIdx = totalItems
	}

	text := fmt.Sprintf(
		"📢 *Управление каналами и группами*\n\n"+
			"Здесь отображаются все подключенные Telegram-каналы и группы.\n"+
			"Вы можете настроить для каждого канала отдельный город, время рассылки, язык и подпись.\n\n"+
			"📊 Всего подключено: *%d*\n"+
			"📄 Страница: *%d из %d*",
		totalItems, page, totalPages,
	)

	var rows [][]tgbotapi.InlineKeyboardButton

	if totalItems == 0 {
		text += "\n\n_Пока нет подключенных каналов или групп._\n\n" +
			"💡 *Как подключить канал:*\n" +
			"1. Добавьте бота в ваш канал как *Администратора* (с правом публикации).\n" +
			"2. Отправьте команду `/channel @username` или перешлите любой пост из канала в этот чат."
	} else {
		pageChats := chats[startIdx:endIdx]
		for _, ch := range pageChats {
			icon := "📢"
			if ch.ChatType == "group" || ch.ChatType == "supergroup" {
				icon = "👥"
			}
			statusIcon := "🔔"
			if len(ch.GetParsedBroadcastTimes()) == 0 {
				statusIcon = "🔕"
			}
			title := ch.Title
			if title == "" {
				title = b.getGroupTitle(ch.ChatID)
			}
			btnText := fmt.Sprintf("%s %s %s (%s)", icon, statusIcon, title, ch.City)
			btn := tgbotapi.NewInlineKeyboardButtonData(btnText, fmt.Sprintf("gback:%d", ch.ChatID))
			rows = append(rows, []tgbotapi.InlineKeyboardButton{btn})
		}
	}

	// Пагинация если элементов больше channelsPageSize
	if totalPages > 1 {
		var navRow []tgbotapi.InlineKeyboardButton
		if page > 1 {
			navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("adm_channels:%d", page-1)))
		} else {
			navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(" ", "noop"))
		}
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d / %d", page, totalPages), "noop"))
		if page < totalPages {
			navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("Вперед ➡️", fmt.Sprintf("adm_channels:%d", page+1)))
		} else {
			navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(" ", "noop"))
		}
		rows = append(rows, navRow)
	}

	// Кнопка подключения нового канала и возврат
	if totalItems > 0 {
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("📋 Дайджест по городам каналов", "adm_channels_digest"),
		})
	}

	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("➕ Подключить новый канал", "adm_add_channel"),
	})

	if isSuper {
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("🔙 В админ-панель", "adm_main"),
		})
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	b.sendOrEditMessage(userChatID, messageID, text, &keyboard)
}

// handleChannelsDigest собирает уникальные города подконтрольных каналов и отправляет расписания с кнопками WhatsApp и MAX
func (b *Bot) handleChannelsDigest(userChatID int64, fromID int64) {
	isSuper := b.storage.IsAdmin(fromID, b.configAdminIDs)
	chats, err := b.storage.GetManagedChats(fromID, isSuper)
	if err != nil || len(chats) == 0 {
		b.sendMessage(userChatID, "❌ *У вас нет подключенных каналов или групп.*")
		return
	}

	citySeen := make(map[string]bool)
	var cities []string
	for _, ch := range chats {
		city := strings.TrimSpace(ch.City)
		if city == "" {
			city = b.defaultCity
		}
		if !citySeen[city] {
			citySeen[city] = true
			cities = append(cities, city)
		}
	}

	if len(cities) == 0 {
		b.sendMessage(userChatID, "❌ *Не удалось определить города для ваших каналов.*")
		return
	}

	now := time.Now()
	headerText := fmt.Sprintf("📋 *Дайджест расписания для городов ваших каналов (%s)*\n\n"+
		"Ниже выведены карточки расписания для всех городов (%d шт.), привязанных к вашим подконтрольным каналам и группам.\n"+
		"Нажмите *«WhatsApp»* или *«MAX»* под нужным городом для быстрой отправки в соответствующий чат:",
		now.Format("02.01.2006"), len(cities))
	b.sendMessage(userChatID, headerText)

	for _, city := range cities {
		item, err := b.client.FetchPrayerTimes(city, now)
		if err != nil {
			log.Printf("Ошибка получения времени намаза для %s в дайджесте каналов: %v", city, err)
			continue
		}
		msgText := api.FormatMessage(item, city, now)
		b.sendMessageWithShare(userChatID, msgText)
	}
}

// handleChannelCommand обрабатывает команду /channel в личных сообщениях
func (b *Bot) handleChannelCommand(userChatID int64, fromID int64, parts []string) {
	if len(parts) < 2 {
		b.handleMyChannelsList(userChatID, fromID, 1, 0)
		return
	}

	input := strings.TrimSpace(parts[1])
	chat, err := b.getChatByUsernameOrID(input)
	if err != nil || chat == nil {
		b.sendMessage(userChatID, "❌ *Не удалось найти канал.*\n\nУбедитесь, что бот уже добавлен в этот канал администратором, и что вы правильно указали `@username` канала.")
		return
	}

	if !b.isGroupAdmin(chat.ID, fromID) {
		b.sendMessage(userChatID, "⛔️ *Доступ ограничен.*\n\nВы не являетесь администратором данного канала.")
		return
	}

	title := chat.Title
	if title == "" {
		title = fmt.Sprintf("Канал %d", chat.ID)
	}

	_ = b.storage.SaveChatInfo(chat.ID, title, chat.Type, fromID)
	b.handleGroupSettings(userChatID, chat.ID, title, 0)
}

// getChatByUsernameOrID ищет чат или канал по username (@channel) или ID (-100...)
func (b *Bot) getChatByUsernameOrID(input string) (*tgbotapi.Chat, error) {
	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, "@") {
		chat, err := b.api.GetChat(tgbotapi.ChatInfoConfig{
			ChatConfig: tgbotapi.ChatConfig{
				SuperGroupUsername: input,
			},
		})
		if err == nil {
			return &chat, nil
		}
	}

	var chatID int64
	if _, err := fmt.Sscanf(input, "%d", &chatID); err == nil && chatID != 0 {
		chat, err := b.api.GetChat(tgbotapi.ChatInfoConfig{
			ChatConfig: tgbotapi.ChatConfig{
				ChatID: chatID,
			},
		})
		if err == nil {
			return &chat, nil
		}
	}

	return nil, fmt.Errorf("чат не найден: %s", input)
}
