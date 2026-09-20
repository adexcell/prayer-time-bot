package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"namaz-time-bot/internal/api"
)

// isGroupAdmin проверяет, является ли пользователь администратором/создателем группы или супер-админом бота
func (b *Bot) isGroupAdmin(groupID int64, userID int64) bool {
	if groupID > 0 {
		return true
	}

	// 1. Проверяем глобальных администраторов бота
	if b.storage.IsAdmin(userID, b.configAdminIDs) {
		return true
	}

	// 2. Проверяем статус в Telegram группе
	member, err := b.api.GetChatMember(tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			ChatID: groupID,
			UserID: userID,
		},
	})
	if err != nil {
		log.Printf("Ошибка проверки прав администратора для groupID %d, userID %d: %v", groupID, userID, err)
		return false
	}

	return member.IsCreator() || member.IsAdministrator()
}

// getGroupTitle возвращает название группы или дефолтное имя
func (b *Bot) getGroupTitle(groupID int64) string {
	chatConfig := tgbotapi.ChatInfoConfig{
		ChatConfig: tgbotapi.ChatConfig{
			ChatID: groupID,
		},
	}
	chatInfo, err := b.api.GetChat(chatConfig)
	if err == nil && chatInfo.Title != "" {
		return chatInfo.Title
	}
	return fmt.Sprintf("Группа %d", groupID)
}

// handleGroupSettingsRedirect отправляет в группу кнопку перехода в ЛС для настройки с автоудалением
func (b *Bot) handleGroupSettingsRedirect(chatID int64, messageID int) {
	// 1. Пытаемся удалить команду пользователя, чтобы не засорять чат (если у бота есть права на удаление)
	if messageID > 0 {
		_, _ = b.api.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
	}

	botUser := b.api.Self.UserName
	url := fmt.Sprintf("https://t.me/%s?start=grp_%d", botUser, chatID)

	text := "⚙️ *Настройки бота для этой группы*\n\n" +
		"Выбор населенного пункта и параметров утренней рассылки для группы настраивается в личных сообщениях с ботом, чтобы не засорять чат.\n\n" +
		"👇 Нажмите кнопку ниже для перехода в настройки (доступно администраторам):\n" +
		"_Это сообщение автоматически исчезнет через 30 секунд._"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("⚙️ Настроить бота в ЛС", url),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✕ Закрыть", fmt.Sprintf("gclose:%d", chatID)),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	sentMsg, err := b.api.Send(msg)
	if err == nil {
		// Автоудаление через 30 секунд
		go func(cID int64, mID int) {
			time.Sleep(30 * time.Second)
			_, _ = b.api.Request(tgbotapi.NewDeleteMessage(cID, mID))
		}(chatID, sentMsg.MessageID)
	} else {
		log.Printf("Ошибка отправки редиректа настроек группы %d: %v", chatID, err)
	}
}

// handleGroupSettings выводит панель управления настройками конкретной группы в ЛС
func (b *Bot) handleGroupSettings(userChatID int64, targetGroupID int64, groupTitle string, messageID int) {
	u, _ := b.storage.GetUser(targetGroupID)
	currentCity, _ := b.storage.GetUserCity(targetGroupID, b.defaultCity)

	selectedTime := "06:00"
	eveningTime := ""
	notify15min := true
	notifyAtTime := true
	isSubscribed := false

	if u != nil {
		isSubscribed = true
		if u.DailyScheduleTime != "" {
			selectedTime = u.DailyScheduleTime
		}
		eveningTime = u.EveningScheduleTime
		notify15min = u.Notify15Min
		notifyAtTime = u.NotifyAtTime
	}

	subStatus := "🔕 Отключена"
	if isSubscribed {
		subStatus = "🔔 Включена"
	}

	eveningDisplay := "Отключена"
	if eveningTime != "" {
		eveningDisplay = eveningTime
	}

	safeTitle := escapeMarkdown(groupTitle)
	safeCity := escapeMarkdown(currentCity)

	text := fmt.Sprintf(
		"👥 *Управление настройками группы*\n"+
			"Чат: *%s*\n\n"+
			"📍 *Населенный пункт:* %s\n"+
			"🌅 *Утренняя рассылка (на день):* %s\n"+
			"🌙 *Вечерняя рассылка (на завтра):* %s\n"+
			"📢 *Статус рассылки в группу:* %s\n\n"+
			"Выберите параметр для изменения:",
		safeTitle, safeCity, selectedTime, eveningDisplay, subStatus,
	)

	// 1. Утреннее время
	times := []string{"05:00", "06:00", "07:00", "08:00", "09:00", "10:00"}
	var timeRow1 []tgbotapi.InlineKeyboardButton
	var timeRow2 []tgbotapi.InlineKeyboardButton

	for i, t := range times {
		label := t
		if t == selectedTime {
			label = "✓ " + t
		}
		btn := tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("gtime:%d:%s", targetGroupID, t))
		if i < 3 {
			timeRow1 = append(timeRow1, btn)
		} else {
			timeRow2 = append(timeRow2, btn)
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
		btn := tgbotapi.NewInlineKeyboardButtonData(display, fmt.Sprintf("getime:%d:%s", targetGroupID, et.val))
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

	labelSub := "📢 Рассылка: [Вкл]"
	if isSubscribed {
		labelSub = "📢 Рассылка: [✓]"
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏙 Изменить город / район", fmt.Sprintf("gloc:%d:city:1", targetGroupID)),
		),
		timeRow1,
		timeRow2,
		eveningRow1,
		eveningRow2,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label15, fmt.Sprintf("gtog15:%d", targetGroupID)),
			tgbotapi.NewInlineKeyboardButtonData(labelAtTime, fmt.Sprintf("gtogat:%d", targetGroupID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(labelSub, fmt.Sprintf("gtogsub:%d", targetGroupID)),
			tgbotapi.NewInlineKeyboardButtonData("📤 Опубликовать пост", fmt.Sprintf("gpost:%d", targetGroupID)),
		),
	)

	if messageID > 0 {
		editMsg := tgbotapi.NewEditMessageText(userChatID, messageID, text)
		editMsg.ParseMode = "Markdown"
		editMsg.ReplyMarkup = &keyboard
		if _, err := b.api.Send(editMsg); err != nil && !strings.Contains(err.Error(), "message is not modified") {
			log.Printf("Ошибка редактирования handleGroupSettings: %v", err)
		}
	} else {
		msg := tgbotapi.NewMessage(userChatID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard
		if _, err := b.api.Send(msg); err != nil {
			log.Printf("Ошибка отправки handleGroupSettings: %v", err)
		}
	}
}

// handleChooseLocationForGroup выводит список населенных пунктов для настройки группы в ЛС
func (b *Bot) handleChooseLocationForGroup(userChatID int64, targetGroupID int64, groupTitle string, isCityTab bool, page int, messageID int) {
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

	currentCity, _ := b.storage.GetUserCity(targetGroupID, b.defaultCity)

	tabName := "🌆 Города"
	if !isCityTab {
		tabName = "🏡 Районы"
	}

	safeTitle := escapeMarkdown(groupTitle)
	safeCurrentCity := escapeMarkdown(currentCity)
	safeTabName := escapeMarkdown(tabName)

	text := fmt.Sprintf(
		"🏙 *Выбор населенного пункта для группы:*\n"+
			"Чат: *%s*\n\n"+
			"Вкладка: %s\n"+
			"📍 Текущий выбор: %s\n"+
			"📄 Страница: %d из %d",
		safeTitle, safeTabName, safeCurrentCity, page, totalPages,
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
		tgbotapi.NewInlineKeyboardButtonData(citiesTabLabel, fmt.Sprintf("gloc:%d:city:1", targetGroupID)),
		tgbotapi.NewInlineKeyboardButtonData(districtsTabLabel, fmt.Sprintf("gloc:%d:district:1", targetGroupID)),
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
		btn := tgbotapi.NewInlineKeyboardButtonData(btnText, fmt.Sprintf("gcity:%d:%d", targetGroupID, item.ID))
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
	tabType := "city"
	if !isCityTab {
		tabType = "district"
	}

	var navRow []tgbotapi.InlineKeyboardButton
	if page > 1 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("gloc:%d:%s:%d", targetGroupID, tabType, page-1)))
	} else {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(" ", "noop"))
	}

	navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d / %d", page, totalPages), "noop"))

	if page < totalPages {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("Вперед ➡️", fmt.Sprintf("gloc:%d:%s:%d", targetGroupID, tabType, page+1)))
	} else {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(" ", "noop"))
	}

	rows = append(rows, navRow)

	// 4. Кнопка возврата в меню настроек группы
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("↩️ Назад к настройкам группы", fmt.Sprintf("gback:%d", targetGroupID)),
	})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	if messageID > 0 {
		editMsg := tgbotapi.NewEditMessageText(userChatID, messageID, text)
		editMsg.ParseMode = "Markdown"
		editMsg.ReplyMarkup = &keyboard
		if _, err := b.api.Send(editMsg); err != nil && !strings.Contains(err.Error(), "message is not modified") {
			log.Printf("Ошибка редактирования handleChooseLocationForGroup: %v", err)
		}
	} else {
		msg := tgbotapi.NewMessage(userChatID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard
		if _, err := b.api.Send(msg); err != nil {
			log.Printf("Ошибка отправки handleChooseLocationForGroup: %v", err)
		}
	}
}

// handleGroupCallbacks обрабатывает нажатия инлайн-кнопок управления группой
func (b *Bot) handleGroupCallbacks(cb *tgbotapi.CallbackQuery) bool {
	if cb == nil || cb.Message == nil {
		return false
	}
	data := cb.Data
	userChatID := cb.Message.Chat.ID
	messageID := cb.Message.MessageID
	fromID := int64(0)
	if cb.From != nil {
		fromID = cb.From.ID
	}

	if strings.HasPrefix(data, "gclose:") {
		parts := strings.Split(data, ":")
		if len(parts) >= 2 {
			targetGroupID, _ := strconv.ParseInt(parts[1], 10, 64)
			if targetGroupID != 0 && fromID != 0 && !b.isGroupAdmin(targetGroupID, fromID) {
				b.api.Send(tgbotapi.NewCallbackWithAlert(cb.ID, "⛔️ Закрыть сообщение могут только администраторы группы."))
				return true
			}
		}
		_, _ = b.api.Request(tgbotapi.NewDeleteMessage(userChatID, messageID))
		b.answerCallback(cb.ID, "")
		return true
	}

	if !strings.HasPrefix(data, "gloc:") &&
		!strings.HasPrefix(data, "gcity:") &&
		!strings.HasPrefix(data, "gtime:") &&
		!strings.HasPrefix(data, "getime:") &&
		!strings.HasPrefix(data, "gtog15:") &&
		!strings.HasPrefix(data, "gtogat:") &&
		!strings.HasPrefix(data, "gtogsub:") &&
		!strings.HasPrefix(data, "gtoday:") &&
		!strings.HasPrefix(data, "gpost:") &&
		!strings.HasPrefix(data, "gback:") {
		return false
	}

	parts := strings.Split(data, ":")
	action := parts[0]
	if len(parts) < 2 {
		return false
	}

	targetGroupID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || targetGroupID == 0 {
		b.answerCallback(cb.ID, "Некорректный ID группы")
		return true
	}

	// Проверяем права администратора в группе
	if !b.isGroupAdmin(targetGroupID, fromID) {
		b.api.Send(tgbotapi.NewCallbackWithAlert(cb.ID, "⛔️ У вас нет прав администратора в этой группе!"))
		return true
	}

	groupTitle := b.getGroupTitle(targetGroupID)

	switch action {
	case "gloc":
		// gloc:<targetGroupID>:<city|district>:<page>
		isCityTab := true
		page := 1
		if len(parts) >= 3 && parts[2] == "district" {
			isCityTab = false
		}
		if len(parts) >= 4 {
			if p, err := strconv.Atoi(parts[3]); err == nil {
				page = p
			}
		}
		b.handleChooseLocationForGroup(userChatID, targetGroupID, groupTitle, isCityTab, page, messageID)
		b.answerCallback(cb.ID, "")

	case "gcity":
		// gcity:<targetGroupID>:<cityID>
		if len(parts) >= 3 {
			cityID, err := strconv.Atoi(parts[2])
			if err == nil && cityID > 0 {
				cityName := ""
				if cityInfo, ok := api.GetCityByID(cityID); ok {
					cityName = cityInfo.DisplayName
				}
				if cityName != "" {
					_ = b.storage.SetUserCity(targetGroupID, cityName)
					b.api.Send(tgbotapi.NewCallbackWithAlert(cb.ID, fmt.Sprintf("✅ Для группы установлено: %s", cityName)))
				}
			}
		}
		b.handleGroupSettings(userChatID, targetGroupID, groupTitle, messageID)

	case "gtime":
		// gtime:<targetGroupID>:<time>
		if len(parts) >= 3 {
			timeStr := parts[2]
			_ = b.storage.UpdateDailyTime(targetGroupID, timeStr)
			b.answerCallback(cb.ID, "Время утренней рассылки: "+timeStr)
		}
		b.handleGroupSettings(userChatID, targetGroupID, groupTitle, messageID)

	case "getime":
		// getime:<targetGroupID>:<time>
		if len(parts) >= 3 {
			timeStr := parts[2]
			if timeStr == "off" {
				_ = b.storage.UpdateEveningTime(targetGroupID, "")
				b.answerCallback(cb.ID, "Вечерняя рассылка отключена")
			} else {
				_ = b.storage.UpdateEveningTime(targetGroupID, timeStr)
				b.answerCallback(cb.ID, "Вечерняя рассылка: "+timeStr)
			}
		}
		b.handleGroupSettings(userChatID, targetGroupID, groupTitle, messageID)

	case "gtog15":
		_ = b.storage.ToggleNotify15min(targetGroupID)
		b.answerCallback(cb.ID, "Настройка напоминания за 15 мин обновлена")
		b.handleGroupSettings(userChatID, targetGroupID, groupTitle, messageID)

	case "gtogat":
		_ = b.storage.ToggleNotifyAtTime(targetGroupID)
		b.answerCallback(cb.ID, "Настройка напоминания в момент намаза обновлена")
		b.handleGroupSettings(userChatID, targetGroupID, groupTitle, messageID)

	case "gtogsub":
		u, _ := b.storage.GetUser(targetGroupID)
		if u != nil {
			_ = b.storage.Unsubscribe(targetGroupID)
			b.api.Send(tgbotapi.NewCallbackWithAlert(cb.ID, "🔕 Ежедневная рассылка в группу отключена"))
		} else {
			city, _ := b.storage.GetUserCity(targetGroupID, b.defaultCity)
			_ = b.storage.Subscribe(targetGroupID, city)
			b.api.Send(tgbotapi.NewCallbackWithAlert(cb.ID, fmt.Sprintf("🔔 Ежедневная рассылка в группу включена (%s)", city)))
		}
		b.handleGroupSettings(userChatID, targetGroupID, groupTitle, messageID)

	case "gtoday":
		city, _ := b.storage.GetUserCity(targetGroupID, b.defaultCity)
		now := time.Now()
		item, err := b.client.FetchPrayerTimes(city, now)
		if err != nil {
			b.sendMessage(userChatID, fmt.Sprintf("К сожалению, не удалось получить расписание для города %s с сервера ДУМ РБ.", city))
		} else {
			msgText := fmt.Sprintf("🕌 *Расписание для «%s» (%s):*\n\n", escapeMarkdown(groupTitle), escapeMarkdown(city)) +
				api.FormatMessage(item, city, now)
			b.sendMessage(userChatID, msgText)
		}
		b.answerCallback(cb.ID, "")

	case "gpost":
		city, _ := b.storage.GetUserCity(targetGroupID, b.defaultCity)
		now := time.Now()
		item, err := b.client.FetchPrayerTimes(city, now)
		if err != nil {
			b.api.Send(tgbotapi.NewCallbackWithAlert(cb.ID, "❌ Ошибка получения расписания от ДУМ РБ"))
		} else {
			msgText := api.FormatMessage(item, city, now)
			msg := tgbotapi.NewMessage(targetGroupID, msgText)
			msg.ParseMode = "Markdown"
			if _, sendErr := b.api.Send(msg); sendErr != nil {
				log.Printf("Ошибка публикации в канал/группу %d: %v", targetGroupID, sendErr)
				b.api.Send(tgbotapi.NewCallbackWithAlert(cb.ID, "❌ Ошибка публикации. Проверьте, что бот является администратором с правом публикации сообщений."))
			} else {
				b.api.Send(tgbotapi.NewCallbackWithAlert(cb.ID, "✅ Расписание успешно опубликовано в канале/группе!"))
			}
		}

	case "gback":
		b.handleGroupSettings(userChatID, targetGroupID, groupTitle, messageID)
		b.answerCallback(cb.ID, "")
	}

	return true
}
