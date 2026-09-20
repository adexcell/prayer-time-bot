package bot

import (
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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
				"👇 Нажмите кнопку ниже, чтобы настроить город и время рассылки:",
			chatTypeLabel, escapeMarkdown(chatTitle),
		)

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⚙️ Настроить расписание", fmt.Sprintf("gback:%d", chatID)),
			),
		)

		msg := tgbotapi.NewMessage(adminID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard
		if _, err := b.api.Send(msg); err != nil {
			log.Printf("Не удалось отправить уведомление админу %d в ЛС: %v", adminID, err)
		}
	}
}

// handleChannelCommand обрабатывает команду /channel в личных сообщениях
func (b *Bot) handleChannelCommand(userChatID int64, fromID int64, parts []string) {
	if len(parts) < 2 {
		helpText := "📢 *Подключение и настройка Telegram-канала*\n\n" +
			"1. Добавьте бота в ваш канал в качестве *Администратора* с правом *«Публикация сообщений»*.\n" +
			"2. Отправьте команду с указанием канала:\n" +
			"   `/channel @username_канала`\n\n" +
			"💡 *Либо просто перешлите сюда любой пост из вашего канала!*"

		b.sendMessage(userChatID, helpText)
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
