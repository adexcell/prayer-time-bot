package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"namaz-time-bot/internal/api"
	"namaz-time-bot/internal/storage"
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

// handleGroupSettings выводит панель управления настройками конкретной группы/канала в ЛС
func (b *Bot) handleGroupSettings(userChatID int64, targetGroupID int64, groupTitle string, messageID int) {
	u, _ := b.storage.GetUser(targetGroupID)
	currentCity, _ := b.storage.GetUserCity(targetGroupID, b.defaultCity)

	notify15min := true
	notifyAtTime := true
	isSubscribed := false
	preset := "ru"
	customFooter := ""
	customHeader := ""
	var broadcastTimes []string

	if u != nil {
		broadcastTimes = u.GetParsedBroadcastTimes()
		if len(broadcastTimes) > 0 {
			isSubscribed = true
		}
		notify15min = u.Notify15Min
		notifyAtTime = u.NotifyAtTime
		if u.PrayerNamesPreset != "" {
			preset = u.PrayerNamesPreset
		}
		customFooter = u.CustomFooter
		customHeader = u.CustomHeader
	}

	subStatus := "🔕 Отключена"
	timesDisplay := "🔕 Отключена"
	if isSubscribed && len(broadcastTimes) > 0 {
		subStatus = "🔔 Включена"
		timesDisplay = strings.Join(broadcastTimes, ", ")
	}

	presetNames := map[string]string{
		"ru":    "Русский",
		"ar":    "العربية (Арабский)",
		"ru_ar": "Русский + Арабский",
		"ba":    "Башкирский / Татарский",
	}
	presetDisplay := presetNames[preset]
	if presetDisplay == "" {
		presetDisplay = "Русский"
	}

	footerDisplay := "По умолчанию"
	if customFooter != "" {
		footerDisplay = customFooter
	}
	headerDisplay := "По умолчанию"
	if customHeader != "" {
		headerDisplay = customHeader
	}

	safeTitle := escapeMarkdown(groupTitle)
	safeCity := escapeMarkdown(currentCity)

	text := fmt.Sprintf(
		"👥 *Управление настройками канала / группы*\n"+
			"Чат: *%s*\n\n"+
			"📍 *Населенный пункт:* %s\n"+
			"⏰ *Времена рассылки:* %s\n"+
			"📢 *Статус рассылки:* %s\n"+
			"🎨 *Стиль названий молитв:* %s\n"+
			"✍️ *Подпись (footer):* %s\n"+
			"📝 *Заголовок:* %s\n\n"+
			"Нажмите *«➕ Добавить время»*, чтобы ввести время (ЧЧ:ММ), или нажмите на крестик у времени для его удаления:",
		safeTitle, safeCity, escapeMarkdown(timesDisplay), subStatus,
		escapeMarkdown(presetDisplay), escapeMarkdown(footerDisplay), escapeMarkdown(headerDisplay),
	)

	var keyboardRows [][]tgbotapi.InlineKeyboardButton

	// 1. Кнопка смены города
	keyboardRows = append(keyboardRows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🏙 Изменить город / район", fmt.Sprintf("gloc:%d:city:1", targetGroupID)),
	})

	// 2. Кнопки удаления имеющихся времен рассылки (по 3 в строке)
	if len(broadcastTimes) > 0 {
		var curRow []tgbotapi.InlineKeyboardButton
		for _, t := range broadcastTimes {
			btn := tgbotapi.NewInlineKeyboardButtonData("✕ "+t, fmt.Sprintf("gdeltime:%d:%s", targetGroupID, t))
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
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить время", fmt.Sprintf("gaddtime:%d", targetGroupID)),
			tgbotapi.NewInlineKeyboardButtonData("🗑 Очистить все", fmt.Sprintf("gcleartimes:%d", targetGroupID)),
		})
	} else {
		// Если времен нет
		keyboardRows = append(keyboardRows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить время рассылки", fmt.Sprintf("gaddtime:%d", targetGroupID)),
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

	labelSub := "📢 Рассылка: [Вкл]"
	if isSubscribed {
		labelSub = "📢 Рассылка: [✓]"
	}

	keyboardRows = append(keyboardRows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(label15, fmt.Sprintf("gtog15:%d", targetGroupID)),
		tgbotapi.NewInlineKeyboardButtonData(labelAtTime, fmt.Sprintf("gtogat:%d", targetGroupID)),
	})
	keyboardRows = append(keyboardRows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(labelSub, fmt.Sprintf("gtogsub:%d", targetGroupID)),
		tgbotapi.NewInlineKeyboardButtonData("📤 Опубликовать пост", fmt.Sprintf("gpost:%d", targetGroupID)),
	})
	keyboardRows = append(keyboardRows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🎨 Оформление и текст", fmt.Sprintf("gstyle:%d", targetGroupID)),
		tgbotapi.NewInlineKeyboardButtonData("👁 Предпросмотр", fmt.Sprintf("gprev:%d", targetGroupID)),
	})
	keyboardRows = append(keyboardRows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🗑 Отключить канал от бота", fmt.Sprintf("gdisconnect:%d", targetGroupID)),
	})
	keyboardRows = append(keyboardRows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔙 К списку каналов", "adm_channels:1"),
	})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)

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

// handleGroupStyleSettings выводит меню настройки текста и оформления рассылки
func (b *Bot) handleGroupStyleSettings(userChatID int64, targetGroupID int64, messageID int) {
	u, _ := b.storage.GetUser(targetGroupID)
	groupTitle := b.getGroupTitle(targetGroupID)

	preset := "ru"
	customFooter := ""
	customHeader := ""
	if u != nil {
		if u.PrayerNamesPreset != "" {
			preset = u.PrayerNamesPreset
		}
		customFooter = u.CustomFooter
		customHeader = u.CustomHeader
	}

	presetNames := map[string]string{
		"ru":    "Русский",
		"ar":    "العربية (Арабский)",
		"ru_ar": "Русский + Арабский (Двуязычный)",
		"ba":    "Башкирский / Татарский",
	}

	footerText := "не задана (по умолчанию)"
	if customFooter != "" {
		footerText = customFooter
	}
	headerText := "по умолчанию (🕌 Расписание намаза)"
	if customHeader != "" {
		headerText = customHeader
	}

	text := fmt.Sprintf(
		"🎨 *Настройка оформления рассылки*\n"+
			"Канал / Группа: *%s*\n\n"+
			"🏷 *Текущий стиль названий:* %s\n"+
			"📝 *Текущий заголовок:* %s\n"+
			"✍️ *Текущая подпись (footer):* %s\n\n"+
			"Выберите стиль названий молитв или настройте текст:",
		escapeMarkdown(groupTitle),
		escapeMarkdown(presetNames[preset]),
		escapeMarkdown(headerText),
		escapeMarkdown(footerText),
	)

	ruLabel := "Русский"
	if preset == "ru" {
		ruLabel = "✓ Русский"
	}
	arLabel := "العربية"
	if preset == "ar" {
		arLabel = "✓ العربية"
	}
	ruArLabel := "Рус + Ар"
	if preset == "ru_ar" {
		ruArLabel = "✓ Рус + Ар"
	}
	baLabel := "Баш / Тат"
	if preset == "ba" {
		baLabel = "✓ Баш / Тат"
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(ruLabel, fmt.Sprintf("gpreset:%d:ru", targetGroupID)),
			tgbotapi.NewInlineKeyboardButtonData(arLabel, fmt.Sprintf("gpreset:%d:ar", targetGroupID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(ruArLabel, fmt.Sprintf("gpreset:%d:ru_ar", targetGroupID)),
			tgbotapi.NewInlineKeyboardButtonData(baLabel, fmt.Sprintf("gpreset:%d:ba", targetGroupID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ Названия молитв", fmt.Sprintf("gprayers:%d", targetGroupID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Изменить заголовок", fmt.Sprintf("gheader:%d", targetGroupID)),
			tgbotapi.NewInlineKeyboardButtonData("✍️ Изменить подпись", fmt.Sprintf("gfooter:%d", targetGroupID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Сбросить оформление", fmt.Sprintf("gresetstyle:%d", targetGroupID)),
			tgbotapi.NewInlineKeyboardButtonData("👁 Предпросмотр", fmt.Sprintf("gprev:%d", targetGroupID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к настройкам канала", fmt.Sprintf("gback:%d", targetGroupID)),
		),
	)

	if messageID > 0 {
		editMsg := tgbotapi.NewEditMessageText(userChatID, messageID, text)
		editMsg.ParseMode = "Markdown"
		editMsg.ReplyMarkup = &keyboard
		if _, err := b.api.Send(editMsg); err != nil && !strings.Contains(err.Error(), "message is not modified") {
			log.Printf("Ошибка редактирования handleGroupStyleSettings: %v", err)
		}
	} else {
		msg := tgbotapi.NewMessage(userChatID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard
		if _, err := b.api.Send(msg); err != nil {
			log.Printf("Ошибка отправки handleGroupStyleSettings: %v", err)
		}
	}
}

func cleanPrayerDisplay(label string) string {
	s := strings.TrimSpace(label)
	s = strings.TrimPrefix(s, "*")
	s = strings.TrimSuffix(s, "*")
	s = strings.TrimSuffix(s, ":")
	return strings.TrimSpace(s)
}

// handleGroupPrayersMenu выводит 2x3 меню для точечной настройки названий молитв
func (b *Bot) handleGroupPrayersMenu(userChatID int64, targetGroupID int64, messageID int) {
	u, _ := b.storage.GetUser(targetGroupID)
	groupTitle := b.getGroupTitle(targetGroupID)

	preset := "ru"
	var customPrayers map[string]string
	if u != nil {
		if u.PrayerNamesPreset != "" {
			preset = u.PrayerNamesPreset
		}
		customPrayers = storage.ParseCustomPrayerNames(u.CustomPrayerNames)
	}

	labels := api.GetPrayerLabels(preset)

	getDisplay := func(key, defaultLabel string) string {
		if custom, ok := customPrayers[key]; ok && strings.TrimSpace(custom) != "" {
			return strings.TrimSpace(custom)
		}
		return cleanPrayerDisplay(defaultLabel)
	}

	fajrLabel := getDisplay("fajr", labels.Fajr)
	sunriseLabel := getDisplay("sunrise", labels.Sunrise)
	dhuhrLabel := getDisplay("dhuhr", labels.Dhuhr)
	asrLabel := getDisplay("asr", labels.Asr)
	maghribLabel := getDisplay("maghrib", labels.Maghrib)
	ishaLabel := getDisplay("isha", labels.Isha)

	text := fmt.Sprintf(
		"✏️ *Настройка названий и эмодзи молитв*\n"+
			"Канал / Группа: *%s*\n\n"+
			"🌅 *Фаджр:* %s\n"+
			"☀️ *Восход:* %s\n"+
			"☀️ *Зухр:* %s\n"+
			"🌤 *Аср:* %s\n"+
			"🌆 *Магриб:* %s\n"+
			"🌙 *Иша:* %s\n\n"+
			"Нажмите на кнопку с молитвой, чтобы точечно изменить её название или эмодзи (поддерживаются любые эмодзи, включая кастомные из Telegram):",
		escapeMarkdown(groupTitle),
		escapeMarkdown(fajrLabel),
		escapeMarkdown(sunriseLabel),
		escapeMarkdown(dhuhrLabel),
		escapeMarkdown(asrLabel),
		escapeMarkdown(maghribLabel),
		escapeMarkdown(ishaLabel),
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fajrLabel, fmt.Sprintf("gprayer:%d:fajr", targetGroupID)),
			tgbotapi.NewInlineKeyboardButtonData(sunriseLabel, fmt.Sprintf("gprayer:%d:sunrise", targetGroupID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(dhuhrLabel, fmt.Sprintf("gprayer:%d:dhuhr", targetGroupID)),
			tgbotapi.NewInlineKeyboardButtonData(asrLabel, fmt.Sprintf("gprayer:%d:asr", targetGroupID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(maghribLabel, fmt.Sprintf("gprayer:%d:maghrib", targetGroupID)),
			tgbotapi.NewInlineKeyboardButtonData(ishaLabel, fmt.Sprintf("gprayer:%d:isha", targetGroupID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Сбросить названия", fmt.Sprintf("gresetprayers:%d", targetGroupID)),
			tgbotapi.NewInlineKeyboardButtonData("👁 Предпросмотр", fmt.Sprintf("gprev:%d", targetGroupID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад в оформление", fmt.Sprintf("gstyle:%d", targetGroupID)),
		),
	)

	if messageID > 0 {
		editMsg := tgbotapi.NewEditMessageText(userChatID, messageID, text)
		editMsg.ParseMode = "Markdown"
		editMsg.ReplyMarkup = &keyboard
		if _, err := b.api.Send(editMsg); err != nil && !strings.Contains(err.Error(), "message is not modified") {
			log.Printf("Ошибка редактирования handleGroupPrayersMenu: %v", err)
		}
	} else {
		msg := tgbotapi.NewMessage(userChatID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard
		if _, err := b.api.Send(msg); err != nil {
			log.Printf("Ошибка отправки handleGroupPrayersMenu: %v", err)
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
		!strings.HasPrefix(data, "gaddtime:") &&
		!strings.HasPrefix(data, "gdeltime:") &&
		!strings.HasPrefix(data, "gcleartimes:") &&
		!strings.HasPrefix(data, "gtog15:") &&
		!strings.HasPrefix(data, "gtogat:") &&
		!strings.HasPrefix(data, "gtogsub:") &&
		!strings.HasPrefix(data, "gtoday:") &&
		!strings.HasPrefix(data, "gpost:") &&
		!strings.HasPrefix(data, "gprev:") &&
		!strings.HasPrefix(data, "gstyle:") &&
		!strings.HasPrefix(data, "gpreset:") &&
		!strings.HasPrefix(data, "gprayers:") &&
		!strings.HasPrefix(data, "gprayer:") &&
		!strings.HasPrefix(data, "gresetprayers:") &&
		!strings.HasPrefix(data, "gheader:") &&
		!strings.HasPrefix(data, "gfooter:") &&
		!strings.HasPrefix(data, "gresetstyle:") &&
		!strings.HasPrefix(data, "gdisconnect:") &&
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

	case "gaddtime":
		b.setUserState(fromID, userState{
			action:        pendingActionBroadcastTime,
			targetGroupID: targetGroupID,
		})
		prompt := fmt.Sprintf(
			"⏰ *Введите время для рассылки в «%s» в формате ЧЧ:ММ*\n\n"+
				"Например: `06:30` или `19:00`.\n\n"+
				"Вы можете настроить любое количество времён рассылки. Отправьте время ответным сообщением (или `/cancel` для отмены).",
			escapeMarkdown(groupTitle),
		)
		b.sendMessage(userChatID, prompt)
		b.answerCallback(cb.ID, "")

	case "gdeltime":
		// gdeltime:<targetGroupID>:<time>
		if len(parts) >= 3 {
			timeStr := strings.Join(parts[2:], ":")
			_ = b.storage.RemoveBroadcastTime(targetGroupID, timeStr)
			b.answerCallback(cb.ID, "Время "+timeStr+" удалено")
		}
		b.handleGroupSettings(userChatID, targetGroupID, groupTitle, messageID)

	case "gcleartimes":
		_ = b.storage.ClearBroadcastTimes(targetGroupID)
		b.answerCallback(cb.ID, "Все времена рассылки очищены")
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
		if u != nil && len(u.GetParsedBroadcastTimes()) > 0 {
			_ = b.storage.Unsubscribe(targetGroupID)
			b.api.Send(tgbotapi.NewCallbackWithAlert(cb.ID, "🔕 Рассылка отключена"))
		} else {
			city, _ := b.storage.GetUserCity(targetGroupID, b.defaultCity)
			_ = b.storage.Subscribe(targetGroupID, city)
			b.api.Send(tgbotapi.NewCallbackWithAlert(cb.ID, fmt.Sprintf("🔔 Рассылка включена (%s)", city)))
		}
		b.handleGroupSettings(userChatID, targetGroupID, groupTitle, messageID)

	case "gtoday":
		city, _ := b.storage.GetUserCity(targetGroupID, b.defaultCity)
		now := time.Now()
		item, err := b.client.FetchPrayerTimes(city, now)
		if err != nil {
			b.sendMessage(userChatID, fmt.Sprintf("К сожалению, не удалось получить расписание для города %s с сервера ДУМ РБ.", city))
		} else {
			u, _ := b.storage.GetUser(targetGroupID)
			cfg := api.PrayerFormatConfig{Preset: "ru"}
			if u != nil {
				cfg.Preset = u.PrayerNamesPreset
				cfg.CustomHeader = u.CustomHeader
				cfg.CustomFooter = u.CustomFooter
				cfg.CustomPrayers = storage.ParseCustomPrayerNames(u.CustomPrayerNames)
			}
			msgText := fmt.Sprintf("🕌 *Расписание для «%s» (%s):*\n\n", escapeMarkdown(groupTitle), escapeMarkdown(city)) +
				api.FormatMessageCustom(item, city, now, cfg)
			b.sendMessage(userChatID, msgText)
		}
		b.answerCallback(cb.ID, "")

	case "gprev":
		city, _ := b.storage.GetUserCity(targetGroupID, b.defaultCity)
		now := time.Now()
		item, err := b.client.FetchPrayerTimes(city, now)
		if err != nil {
			b.sendMessage(userChatID, fmt.Sprintf("❌ Не удалось получить расписание для города %s от ДУМ РБ.", city))
		} else {
			u, _ := b.storage.GetUser(targetGroupID)
			cfg := api.PrayerFormatConfig{Preset: "ru"}
			if u != nil {
				cfg.Preset = u.PrayerNamesPreset
				cfg.CustomHeader = u.CustomHeader
				cfg.CustomFooter = u.CustomFooter
				cfg.CustomPrayers = storage.ParseCustomPrayerNames(u.CustomPrayerNames)
			}
			previewText := fmt.Sprintf("👁 *Предпросмотр рассылки для «%s»:*\n\n", escapeMarkdown(groupTitle)) +
				api.FormatMessageCustom(item, city, now, cfg)

			// Удаляем предыдущее превью сообщение пользователя при повторных нажатиях
			if prevID := b.getAndClearPreviewMessage(userChatID); prevID > 0 {
				_, _ = b.api.Request(tgbotapi.NewDeleteMessage(userChatID, prevID))
			}

			msg := tgbotapi.NewMessage(userChatID, previewText)
			msg.ParseMode = "Markdown"
			sentMsg, err := b.api.Send(msg)
			if err == nil {
				b.setPreviewMessage(userChatID, sentMsg.MessageID)
			}
		}
		b.answerCallback(cb.ID, "")

	case "gpost":
		city, _ := b.storage.GetUserCity(targetGroupID, b.defaultCity)
		now := time.Now()
		item, err := b.client.FetchPrayerTimes(city, now)
		if err != nil {
			b.api.Send(tgbotapi.NewCallbackWithAlert(cb.ID, "❌ Ошибка получения расписания от ДУМ РБ"))
		} else {
			u, _ := b.storage.GetUser(targetGroupID)
			cfg := api.PrayerFormatConfig{Preset: "ru"}
			if u != nil {
				cfg.Preset = u.PrayerNamesPreset
				cfg.CustomHeader = u.CustomHeader
				cfg.CustomFooter = u.CustomFooter
				cfg.CustomPrayers = storage.ParseCustomPrayerNames(u.CustomPrayerNames)
			}
			msgText := api.FormatMessageCustom(item, city, now, cfg)
			msg := tgbotapi.NewMessage(targetGroupID, msgText)
			msg.ParseMode = "Markdown"
			msg.ReplyMarkup = CreateShareInlineKeyboard(msgText)
			if _, sendErr := b.api.Send(msg); sendErr != nil {
				log.Printf("Ошибка публикации в канал/группу %d: %v", targetGroupID, sendErr)
				b.api.Send(tgbotapi.NewCallbackWithAlert(cb.ID, "❌ Ошибка публикации. Проверьте, что бот назначен администратором канала с правом публикации сообщений."))
			} else {
				b.api.Send(tgbotapi.NewCallbackWithAlert(cb.ID, "✅ Расписание успешно опубликовано в канале/группе!"))
			}
		}

	case "gstyle":
		b.handleGroupStyleSettings(userChatID, targetGroupID, messageID)
		b.answerCallback(cb.ID, "")

	case "gpreset":
		// gpreset:<targetGroupID>:<preset>
		if len(parts) >= 3 {
			preset := parts[2]
			_ = b.storage.UpdateChatPreset(targetGroupID, preset)
			b.answerCallback(cb.ID, "Стиль названий обновлен")
		}
		b.handleGroupStyleSettings(userChatID, targetGroupID, messageID)

	case "gprayers":
		b.handleGroupPrayersMenu(userChatID, targetGroupID, messageID)
		b.answerCallback(cb.ID, "")

	case "gprayer":
		// gprayer:<targetGroupID>:<prayerKey>
		if len(parts) >= 3 {
			prayerKey := parts[2]
			b.setUserState(fromID, userState{
				action:        pendingActionPrayer,
				targetGroupID: targetGroupID,
				prayerKey:     prayerKey,
			})
			prayerNamesRu := map[string]string{
				"fajr":    "Фаджр",
				"sunrise": "Восход",
				"dhuhr":   "Зухр",
				"asr":     "Аср",
				"maghrib": "Магриб",
				"isha":    "Иша",
			}
			nameRu := prayerNamesRu[prayerKey]
			prompt := fmt.Sprintf(
				"✏️ *Введите новое название или эмодзи для «%s» в группе/канале «%s»*\n\n"+
					"Вы можете указать любой текст, стандартные эмодзи или кастомные эмодзи из Telegram.\n"+
					"Например:\n`🌅 Утренний намаз (Фаджр)`\n\n"+
					"Отправьте текст ответным сообщением, либо `-` для возврата к названию из выбранного стиля.",
				nameRu, escapeMarkdown(groupTitle),
			)
			b.sendMessage(userChatID, prompt)
		}
		b.answerCallback(cb.ID, "")

	case "gresetprayers":
		_ = b.storage.ResetCustomPrayerNames(targetGroupID)
		b.api.Send(tgbotapi.NewCallbackWithAlert(cb.ID, "Все названия молитв сброшены к выбранному стилю"))
		b.handleGroupPrayersMenu(userChatID, targetGroupID, messageID)

	case "gheader":
		b.setUserState(fromID, userState{action: pendingActionHeader, targetGroupID: targetGroupID})
		prompt := fmt.Sprintf("📝 *Введите заголовок рассылки для «%s»*\n\nНапример:\n`🕌 Расписание намаза в мечети Ихлас`\n\nОтправьте текст ответным сообщением, либо `-` для возврата к заголовку по умолчанию.", escapeMarkdown(groupTitle))
		b.sendMessage(userChatID, prompt)
		b.answerCallback(cb.ID, "")

	case "gfooter":
		b.setUserState(fromID, userState{action: pendingActionFooter, targetGroupID: targetGroupID})
		prompt := fmt.Sprintf("✍️ *Введите подпись (footer) для «%s»*\n\n"+
			"Этот текст будет добавляться в конце каждого сообщения рассылки.\n\n"+
			"💡 *Примеры оформления:*\n"+
			"• Ссылка текстом (гиперссылка): `📢 [Подписаться на канал](https://t.me/channel_name)`\n"+
			"• Обычный текст: `📢 Мечеть «Ихлас»`\n"+
			"• Тег канала: `📢 Наш канал: @channel_name`\n\n"+
			"Отправьте текст ответным сообщением, либо `-` для удаления подписи.", escapeMarkdown(groupTitle))
		b.sendMessage(userChatID, prompt)
		b.answerCallback(cb.ID, "")

	case "gresetstyle":
		_ = b.storage.ResetChatCustomization(targetGroupID)
		b.answerCallback(cb.ID, "Оформление сброшено по умолчанию")
		b.handleGroupStyleSettings(userChatID, targetGroupID, messageID)

	case "gdisconnect":
		_ = b.storage.DeleteChat(targetGroupID)
		b.api.Send(tgbotapi.NewCallbackWithAlert(cb.ID, "🗑 Канал успешно отключен от рассылки и удален."))
		b.handleMyChannelsList(userChatID, fromID, 1, messageID)

	case "gback":
		b.handleGroupSettings(userChatID, targetGroupID, groupTitle, messageID)
		b.answerCallback(cb.ID, "")
	}

	return true
}
