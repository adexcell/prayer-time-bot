package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"namaz-time-bot/internal/storage"
)

// handleAdminHadithsMenu отображает главное меню управления хадисами
func (b *Bot) handleAdminHadithsMenu(chatID int64, fromID int64, messageID int) {
	if !b.storage.IsAdmin(fromID, b.configAdminIDs) {
		b.sendMessage(chatID, "⛔️ У вас нет прав администратора.")
		return
	}

	total, active, byCat, byCol, _ := b.storage.GetHadithsStats()

	dailyEn := b.storage.GetSetting("hadith_daily_enabled", "1") == "1"
	dailyTime := b.storage.GetSetting("hadith_daily_time", "09:00")

	prayerEn := b.storage.GetSetting("hadith_prayer_enabled", "1") == "1"

	fastingEn := b.storage.GetSetting("hadith_fasting_enabled", "1") == "1"
	fastingTime := b.storage.GetSetting("hadith_fasting_time", "07:00")

	monthsEn := b.storage.GetSetting("hadith_months_enabled", "1") == "1"
	monthsTime := b.storage.GetSetting("hadith_months_time", "12:00")

	formatStatus := func(en bool) string {
		if en {
			return "🟢 Вкл"
		}
		return "🔴 Выкл"
	}

	text := fmt.Sprintf(
		"📖 *Управление хадисами*\n\n"+
			"Сборники: *Сады праведных, Сахих аль-Бухари, Сахих Муслим*\n"+
			"Всего в базе: *%d* (активных: *%d*)\n\n"+
			"📚 *По сборникам:*\n"+
			"• Сахих аль-Бухари: %d\n"+
			"• Сахих Муслим: %d\n"+
			"• Сады праведных: %d\n\n"+
			"⚙️ *Статус рассылок:*\n"+
			"• Ежедневный хадис: %s (%s)\n"+
			"• Хадисы к намазам: %s\n"+
			"• О посте (Пн/Чт и 13-15): %s (%s)\n"+
			"• О лунных месяцах: %s (%s)",
		total, active,
		byCol[storage.CollectionBukhari],
		byCol[storage.CollectionMuslim],
		byCol[storage.CollectionRiyad],
		formatStatus(dailyEn), dailyTime,
		formatStatus(prayerEn),
		formatStatus(fastingEn), fastingTime,
		formatStatus(monthsEn), monthsTime,
	)
	_ = byCat

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки рассылок", "adm_hadith_cfg"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📚 Список хадисов", "adm_hadith_list:all:1"),
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить хадис", "adm_hadith_add_cat"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎲 Случайный хадис", "adm_hadith_test_rnd"),
			tgbotapi.NewInlineKeyboardButtonData("📢 Разослать сейчас", "adm_hadith_bc_confirm"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад в админ-панель", "adm_main"),
		),
	)

	b.sendOrEditMessage(chatID, messageID, text, &keyboard)
}

// handleAdminHadithsConfig меню переключателей и времени рассылок хадисов
func (b *Bot) handleAdminHadithsConfig(chatID int64, fromID int64, messageID int) {
	if !b.storage.IsAdmin(fromID, b.configAdminIDs) {
		return
	}

	dailyEn := b.storage.GetSetting("hadith_daily_enabled", "1") == "1"
	dailyTime := b.storage.GetSetting("hadith_daily_time", "09:00")

	prayerEn := b.storage.GetSetting("hadith_prayer_enabled", "1") == "1"

	fastingEn := b.storage.GetSetting("hadith_fasting_enabled", "1") == "1"
	fastingTime := b.storage.GetSetting("hadith_fasting_time", "07:00")

	monthsEn := b.storage.GetSetting("hadith_months_enabled", "1") == "1"
	monthsTime := b.storage.GetSetting("hadith_months_time", "12:00")

	bukhariEn := b.storage.IsCollectionActive(storage.CollectionBukhari)
	muslimEn := b.storage.IsCollectionActive(storage.CollectionMuslim)
	riyadEn := b.storage.IsCollectionActive(storage.CollectionRiyad)

	formatStatus := func(en bool) string {
		if en {
			return "🟢 Вкл"
		}
		return "🔴 Выкл"
	}

	text := fmt.Sprintf(
		"⚙️ *Настройки показа и рассылок хадисов*\n\n"+
			"Здесь вы можете включать и отключать сборники хадисов, а также настраивать типы и время рассылок:\n\n"+
			"📚 *Сборники хадисов:*\n"+
			"• Сахих аль-Бухари: %s\n"+
			"• Сахих Муслим: %s\n"+
			"• Сады праведных: %s\n\n"+
			"⏰ *Типы рассылок:*\n"+
			"• Хадис дня: %s (%s)\n"+
			"• К намазам: %s\n"+
			"• О посте (Пн/Чт и 13-15): %s (%s)\n"+
			"• О месяцах: %s (%s)",
		formatStatus(bukhariEn),
		formatStatus(muslimEn),
		formatStatus(riyadEn),
		formatStatus(dailyEn), dailyTime,
		formatStatus(prayerEn),
		formatStatus(fastingEn), fastingTime,
		formatStatus(monthsEn), monthsTime,
	)

	btnBukhari := "Бухари: [ ]"
	if bukhariEn {
		btnBukhari = "Бухари: [✓]"
	}
	btnMuslim := "Муслим: [ ]"
	if muslimEn {
		btnMuslim = "Муслим: [✓]"
	}
	btnRiyad := "Сады праведных: [ ]"
	if riyadEn {
		btnRiyad = "Сады праведных: [✓]"
	}

	btnTogDaily := "Хадис дня: [ ]"
	if dailyEn {
		btnTogDaily = "Хадис дня: [✓]"
	}
	btnTogPrayer := "К намазам: [ ]"
	if prayerEn {
		btnTogPrayer = "К намазам: [✓]"
	}
	btnTogFasting := "О посте: [ ]"
	if fastingEn {
		btnTogFasting = "О посте: [✓]"
	}
	btnTogMonths := "О месяцах: [ ]"
	if monthsEn {
		btnTogMonths = "О месяцах: [✓]"
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(btnBukhari, "adm_hadith_tog_col:bukhari"),
			tgbotapi.NewInlineKeyboardButtonData(btnMuslim, "adm_hadith_tog_col:muslim"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(btnRiyad, "adm_hadith_tog_col:riyad"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(btnTogDaily, "adm_hadith_tog:daily"),
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("⏰ Время: %s", dailyTime), "adm_hadith_settime:daily"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(btnTogPrayer, "adm_hadith_tog:prayer"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(btnTogFasting, "adm_hadith_tog:fasting"),
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("⏰ Время: %s", fastingTime), "adm_hadith_settime:fasting"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(btnTogMonths, "adm_hadith_tog:months"),
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("⏰ Время: %s", monthsTime), "adm_hadith_settime:months"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 К управлению хадисами", "adm_hadiths"),
		),
	)

	b.sendOrEditMessage(chatID, messageID, text, &keyboard)
}

// handleAdminHadithsList список хадисов с пагинацией и фильтром
func (b *Bot) handleAdminHadithsList(chatID int64, category string, page int, messageID int) {
	if page < 1 {
		page = 1
	}
	pageSize := 5
	offset := (page - 1) * pageSize

	hadiths, total, err := b.storage.GetHadiths(category, pageSize, offset)
	if err != nil {
		b.sendMessage(chatID, "❌ Ошибка получения списка хадисов.")
		return
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	catTitle := "Все категории"
	if category != "" && category != "all" {
		catTitle = storage.GetCategoryTitle(category)
	}

	text := fmt.Sprintf(
		"📚 *Список хадисов*\n\n"+
			"Категория: *%s*\n"+
			"Всего: *%d* (стр. *%d* из *%d*)\n\n"+
			"Выберите хадис для просмотра или удаления:",
		catTitle, total, page, totalPages,
	)

	var rows [][]tgbotapi.InlineKeyboardButton

	for _, h := range hadiths {
		status := "🟢"
		if !h.IsActive {
			status = "🔴"
		}
		preview := h.Text
		if len([]rune(preview)) > 35 {
			preview = string([]rune(preview)[:35]) + "..."
		}
		btnLabel := fmt.Sprintf("%s #%d %s", status, h.ID, preview)
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(btnLabel, fmt.Sprintf("adm_hadith_view:%d", h.ID)),
		})
	}

	// Пагинация
	var navRow []tgbotapi.InlineKeyboardButton
	if page > 1 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("adm_hadith_list:%s:%d", category, page-1)))
	} else {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(" ", "noop"))
	}
	navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d / %d", page, totalPages), "noop"))
	if page < totalPages {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("Вперед ➡️", fmt.Sprintf("adm_hadith_list:%s:%d", category, page+1)))
	} else {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(" ", "noop"))
	}
	rows = append(rows, navRow)

	// Фильтр по категориям
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔍 Фильтр по категории", "adm_hadith_cat_filter"),
	})

	// Назад
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔙 К меню хадисов", "adm_hadiths"),
	})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	b.sendOrEditMessage(chatID, messageID, text, &keyboard)
}

// handleAdminHadithView детальный просмотр хадиса
func (b *Bot) handleAdminHadithView(chatID int64, hadithID int64, messageID int) {
	h, err := b.storage.GetHadithByID(hadithID)
	if err != nil || h == nil {
		b.sendMessage(chatID, "❌ Хадис не найден.")
		return
	}

	statusStr := "🟢 Активен (участвует в рассылках)"
	togBtnLabel := "🔴 Отключить показ"
	if !h.IsActive {
		statusStr = "🔴 Отключен"
		togBtnLabel = "🟢 Включить показ"
	}

	text := fmt.Sprintf(
		"📖 *Хадис #%d*\n\n"+
			"«%s»\n\n"+
			"📚 *Источник:* %s\n"+
			"📁 *Сборник:* %s\n"+
			"🏷 *Категория:* %s\n"+
			"Статус: %s",
		h.ID, h.Text, h.Source,
		storage.GetCollectionTitle(h.Collection),
		storage.GetCategoryTitle(h.Category),
		statusStr,
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(togBtnLabel, fmt.Sprintf("adm_hadith_tog_active:%d", h.ID)),
			tgbotapi.NewInlineKeyboardButtonData("🗑 Удалить", fmt.Sprintf("adm_hadith_del:%d", h.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 К списку хадисов", "adm_hadith_list:all:1"),
		),
	)

	b.sendOrEditMessage(chatID, messageID, text, &keyboard)
}

// handleAdminHadithAddCat шаг 1: выбор категории для добавления
func (b *Bot) handleAdminHadithAddCat(chatID int64, messageID int) {
	text := "➕ *Добавление хадиса — Шаг 1 из 3*\n\nВыберите категорию хадиса:"

	var rows [][]tgbotapi.InlineKeyboardButton
	var currentRow []tgbotapi.InlineKeyboardButton

	for _, cat := range storage.AllCategories {
		btn := tgbotapi.NewInlineKeyboardButtonData(storage.GetCategoryTitle(cat), fmt.Sprintf("adm_hadith_add_col:%s", cat))
		currentRow = append(currentRow, btn)
		if len(currentRow) == 2 {
			rows = append(rows, currentRow)
			currentRow = []tgbotapi.InlineKeyboardButton{}
		}
	}
	if len(currentRow) > 0 {
		rows = append(rows, currentRow)
	}

	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔙 Отмена", "adm_hadiths"),
	})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	b.sendOrEditMessage(chatID, messageID, text, &keyboard)
}

// handleAdminHadithAddCol шаг 2: выбор сборника
func (b *Bot) handleAdminHadithAddCol(chatID int64, category string, messageID int) {
	text := fmt.Sprintf(
		"➕ *Добавление хадиса — Шаг 2 из 3*\n\n"+
			"Категория: *%s*\n\n"+
			"Выберите сборник хадисов:",
		storage.GetCategoryTitle(category),
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Сады праведных", fmt.Sprintf("adm_hadith_prompt:%s:%s", category, storage.CollectionRiyad)),
			tgbotapi.NewInlineKeyboardButtonData("Сахих аль-Бухари", fmt.Sprintf("adm_hadith_prompt:%s:%s", category, storage.CollectionBukhari)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Сахих Муслим", fmt.Sprintf("adm_hadith_prompt:%s:%s", category, storage.CollectionMuslim)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "adm_hadith_add_cat"),
		),
	)

	b.sendOrEditMessage(chatID, messageID, text, &keyboard)
}

// handleAdminBroadcastConfirm подтверждение моментальной рассылки
func (b *Bot) handleAdminBroadcastConfirm(chatID int64, messageID int) {
	text := "📢 *Ручная рассылка хадиса дня*\n\n" +
		"Вы действительно хотите выбрать случайный хадис и разослать его прямо сейчас всем подписчикам бота?"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Да, разослать сейчас", "adm_hadith_bc_now"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "adm_hadiths"),
		),
	)

	b.sendOrEditMessage(chatID, messageID, text, &keyboard)
}

// handleAdminBroadcastNow выполняет рассылку случайного хадиса дня всем подписчикам
func (b *Bot) handleAdminBroadcastNow(chatID int64, fromID int64, messageID int) {
	if !b.storage.IsAdmin(fromID, b.configAdminIDs) {
		return
	}

	h, err := b.storage.GetRandomHadithAny()
	if err != nil || h == nil {
		b.sendMessage(chatID, "❌ Не удалось найти активный хадис для рассылки.")
		return
	}

	subscribers, err := b.storage.GetSubscribers()
	if err != nil || len(subscribers) == 0 {
		b.sendMessage(chatID, "❌ Список подписчиков пуст.")
		return
	}

	msg := fmt.Sprintf("📖 *Хадис дня*\n\n«%s»\n\n📚 *Источник:* %s", h.Text, h.Source)
	sentCount := 0

	for _, user := range subscribers {
		if user.HadithDailyEnabled {
			b.sendMessageWithShare(user.ChatID, msg)
			sentCount++
			time.Sleep(25 * time.Millisecond)
		}
	}

	b.sendMessage(chatID, fmt.Sprintf("✅ Хадис дня успешно разослан %d подписчикам!", sentCount))
	b.handleAdminHadithsMenu(chatID, fromID, 0)
}

// handleAdminHadithsCallbacks маршрутизирует все callback-запросы управления хадисами
func (b *Bot) handleAdminHadithsCallbacks(cb *tgbotapi.CallbackQuery) bool {
	if cb == nil || cb.Message == nil {
		return false
	}
	data := cb.Data
	chatID := cb.Message.Chat.ID
	messageID := cb.Message.MessageID
	fromID := int64(0)
	if cb.From != nil {
		fromID = cb.From.ID
	}

	if !strings.HasPrefix(data, "adm_hadith") {
		return false
	}

	if !b.storage.IsAdmin(fromID, b.configAdminIDs) {
		b.answerCallback(cb.ID, "⛔️ Доступ запрещен.")
		return true
	}

	switch {
	case data == "adm_hadiths":
		b.handleAdminHadithsMenu(chatID, fromID, messageID)
		b.answerCallback(cb.ID, "")

	case data == "adm_hadith_cfg":
		b.handleAdminHadithsConfig(chatID, fromID, messageID)
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "adm_hadith_tog_col:"):
		col := strings.TrimPrefix(data, "adm_hadith_tog_col:")
		_ = b.storage.ToggleCollection(col)
		b.handleAdminHadithsConfig(chatID, fromID, messageID)
		b.answerCallback(cb.ID, "Статус сборника изменен")

	case strings.HasPrefix(data, "adm_hadith_tog:"):
		feature := strings.TrimPrefix(data, "adm_hadith_tog:")
		switch feature {
		case "daily":
			cur := b.storage.GetSetting("hadith_daily_enabled", "1")
			if cur == "1" {
				_ = b.storage.SetSetting("hadith_daily_enabled", "0")
			} else {
				_ = b.storage.SetSetting("hadith_daily_enabled", "1")
			}
		case "prayer":
			cur := b.storage.GetSetting("hadith_prayer_enabled", "1")
			if cur == "1" {
				_ = b.storage.SetSetting("hadith_prayer_enabled", "0")
			} else {
				_ = b.storage.SetSetting("hadith_prayer_enabled", "1")
			}
		case "fasting":
			cur := b.storage.GetSetting("hadith_fasting_enabled", "1")
			if cur == "1" {
				_ = b.storage.SetSetting("hadith_fasting_enabled", "0")
			} else {
				_ = b.storage.SetSetting("hadith_fasting_enabled", "1")
			}
		case "months":
			cur := b.storage.GetSetting("hadith_months_enabled", "1")
			if cur == "1" {
				_ = b.storage.SetSetting("hadith_months_enabled", "0")
			} else {
				_ = b.storage.SetSetting("hadith_months_enabled", "1")
			}
		}
		b.handleAdminHadithsConfig(chatID, fromID, messageID)
		b.answerCallback(cb.ID, "Статус изменен")

	case strings.HasPrefix(data, "adm_hadith_settime:"):
		feature := strings.TrimPrefix(data, "adm_hadith_settime:")
		settingKey := "hadith_daily_time"
		featureTitle := "ежедневного хадиса"
		switch feature {
		case "fasting":
			settingKey = "hadith_fasting_time"
			featureTitle = "хадисов о посте (Пн/Чт и Белые дни)"
		case "months":
			settingKey = "hadith_months_time"
			featureTitle = "хадисов о лунных месяцах"
		}

		b.setUserState(fromID, userState{
			action:     pendingActionHadithTime,
			settingKey: settingKey,
		})

		prompt := fmt.Sprintf(
			"⏰ *Установка времени рассылки %s*\n\n"+
				"Введите время в формате `ЧЧ:ММ` (например: `09:00` или `06:30`):\n\n"+
				"Отправьте `-` для отмены.",
			featureTitle,
		)
		b.sendMessage(chatID, prompt)
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "adm_hadith_list:"):
		parts := strings.Split(strings.TrimPrefix(data, "adm_hadith_list:"), ":")
		cat := "all"
		page := 1
		if len(parts) >= 1 && parts[0] != "" {
			cat = parts[0]
		}
		if len(parts) >= 2 {
			page, _ = strconv.Atoi(parts[1])
		}
		b.handleAdminHadithsList(chatID, cat, page, messageID)
		b.answerCallback(cb.ID, "")

	case data == "adm_hadith_cat_filter":
		text := "🔍 *Выберите категорию для фильтрации:*"
		var rows [][]tgbotapi.InlineKeyboardButton
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("🌐 Все категории", "adm_hadith_list:all:1"),
		})
		var curRow []tgbotapi.InlineKeyboardButton
		for _, cat := range storage.AllCategories {
			btn := tgbotapi.NewInlineKeyboardButtonData(storage.GetCategoryTitle(cat), fmt.Sprintf("adm_hadith_list:%s:1", cat))
			curRow = append(curRow, btn)
			if len(curRow) == 2 {
				rows = append(rows, curRow)
				curRow = []tgbotapi.InlineKeyboardButton{}
			}
		}
		if len(curRow) > 0 {
			rows = append(rows, curRow)
		}
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("🔙 К списку", "adm_hadith_list:all:1"),
		})
		kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
		b.sendOrEditMessage(chatID, messageID, text, &kb)
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "adm_hadith_view:"):
		idStr := strings.TrimPrefix(data, "adm_hadith_view:")
		hID, _ := strconv.ParseInt(idStr, 10, 64)
		b.handleAdminHadithView(chatID, hID, messageID)
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "adm_hadith_tog_active:"):
		idStr := strings.TrimPrefix(data, "adm_hadith_tog_active:")
		hID, _ := strconv.ParseInt(idStr, 10, 64)
		_ = b.storage.ToggleHadithActive(hID)
		b.handleAdminHadithView(chatID, hID, messageID)
		b.answerCallback(cb.ID, "Статус хадиса обновлен")

	case strings.HasPrefix(data, "adm_hadith_del:"):
		idStr := strings.TrimPrefix(data, "adm_hadith_del:")
		hID, _ := strconv.ParseInt(idStr, 10, 64)
		_ = b.storage.DeleteHadith(hID)
		b.handleAdminHadithsList(chatID, "all", 1, messageID)
		b.answerCallback(cb.ID, "Хадис удален")

	case data == "adm_hadith_add_cat":
		b.handleAdminHadithAddCat(chatID, messageID)
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "adm_hadith_add_col:"):
		cat := strings.TrimPrefix(data, "adm_hadith_add_col:")
		b.handleAdminHadithAddCol(chatID, cat, messageID)
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "adm_hadith_prompt:"):
		parts := strings.Split(strings.TrimPrefix(data, "adm_hadith_prompt:"), ":")
		if len(parts) >= 2 {
			cat := parts[0]
			col := parts[1]
			b.setUserState(fromID, userState{
				action:    pendingActionHadithText,
				hadithCat: cat,
				hadithCol: col,
			})
			prompt := fmt.Sprintf(
				"📝 *Добавление хадиса — Ввод текста*\n\n"+
					"Категория: *%s*\n"+
					"Сборник: *%s*\n\n"+
					"Отправьте текст хадиса следующим сообщением в этот чат.\n"+
					"Или отправьте `-` для отмены.",
				storage.GetCategoryTitle(cat),
				storage.GetCollectionTitle(col),
			)
			b.sendMessage(chatID, prompt)
		}
		b.answerCallback(cb.ID, "")

	case data == "adm_hadith_test_rnd":
		h, err := b.storage.GetRandomHadithAny()
		if err != nil || h == nil {
			b.answerCallback(cb.ID, "❌ Нет активных хадисов")
			return true
		}
		msg := fmt.Sprintf(
			"🎲 *Тестовый предпросмотр случайного хадиса:*\n\n"+
				"«%s»\n\n"+
				"📚 *Источник:* %s\n"+
				"📁 *Сборник:* %s\n"+
				"🏷 *Категория:* %s",
			h.Text, h.Source,
			storage.GetCollectionTitle(h.Collection),
			storage.GetCategoryTitle(h.Category),
		)
		b.sendMessage(chatID, msg)
		b.answerCallback(cb.ID, "Хадис отправлен в чат")

	case data == "adm_hadith_bc_confirm":
		b.handleAdminBroadcastConfirm(chatID, messageID)
		b.answerCallback(cb.ID, "")

	case data == "adm_hadith_bc_now":
		b.handleAdminBroadcastNow(chatID, fromID, messageID)
		b.answerCallback(cb.ID, "")
	}

	return true
}
