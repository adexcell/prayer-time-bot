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

// handleAdmin открывает главное меню панели администратора
func (b *Bot) handleAdmin(chatID int64, fromID int64, messageID int) {
	if !b.storage.IsAdmin(fromID, b.configAdminIDs) {
		b.sendMessage(chatID, "⛔️ У вас нет прав администратора.")
		return
	}

	text := "🛠 *Панель администратора*\n\n" +
		"Здесь вы можете управлять подключенными каналами и группами, временем молитв для городов и районов РБ, настраивать срок действия корректировок, а также управлять администраторами бота."

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📢 Каналы и группы", "adm_channels:1"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить корректировку", "adm_add_rule:city:1"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Активные правила", "adm_rules"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👥 Администраторы", "adm_admins"),
			tgbotapi.NewInlineKeyboardButtonData("📊 Статистика", "adm_stats"),
		),
	)

	b.sendOrEditMessage(chatID, messageID, text, &keyboard)
}

// handleAdminAddRuleCityStep показывает выбор населенного пункта для корректировки
func (b *Bot) handleAdminAddRuleCityStep(chatID int64, isCityTab bool, page int, messageID int) {
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

	tabName := "🌆 Города"
	if !isCityTab {
		tabName = "🏡 Районы"
	}

	text := fmt.Sprintf(
		"➕ *Шаг 1 из 4: Выберите населенный пункт*\n\n"+
			"Вкладка: %s\n"+
			"Страница: %d из %d",
		tabName, page, totalPages,
	)

	var rows [][]tgbotapi.InlineKeyboardButton

	// 1. Вкладки
	citiesTabLabel := "🌆 Города (21)"
	districtsTabLabel := "🏡 Районы (40)"
	if isCityTab {
		citiesTabLabel = "🔹 🌆 Города (21)"
	} else {
		districtsTabLabel = "🔹 🏡 Районы (40)"
	}

	tabRow := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(citiesTabLabel, "adm_add_rule:city:1"),
		tgbotapi.NewInlineKeyboardButtonData(districtsTabLabel, "adm_add_rule:district:1"),
	}
	rows = append(rows, tabRow)

	// 2. Список городов
	pageItems := items[startIdx:endIdx]
	var currentRow []tgbotapi.InlineKeyboardButton
	for _, item := range pageItems {
		btn := tgbotapi.NewInlineKeyboardButtonData(item.DisplayName, fmt.Sprintf("adm_sel_city:%d", item.ID))
		currentRow = append(currentRow, btn)
		if len(currentRow) == 2 {
			rows = append(rows, currentRow)
			currentRow = []tgbotapi.InlineKeyboardButton{}
		}
	}
	if len(currentRow) > 0 {
		rows = append(rows, currentRow)
	}

	// 3. Пагинация
	pagePrefix := "adm_add_rule:city"
	if !isCityTab {
		pagePrefix = "adm_add_rule:district"
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

	// 4. Кнопка возврата в меню
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад в админ-панель", "adm_main"),
	})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	b.sendOrEditMessage(chatID, messageID, text, &keyboard)
}

// handleAdminSelectPrayerStep выбор молитвы для корректировки
func (b *Bot) handleAdminSelectPrayerStep(chatID int64, cityID int, messageID int) {
	cityInfo, ok := api.GetCityByID(cityID)
	cityName := "Уфа"
	if ok {
		cityName = cityInfo.DisplayName
	}

	text := fmt.Sprintf(
		"➕ *Шаг 2 из 4: Выберите молитву*\n\n"+
			"📍 Населенный пункт: *%s*\n\n"+
			"Выберите молитву, для которой хотите изменить время:",
		cityName,
	)

	prayers := []struct {
		Label string
		Key   string
	}{
		{"🌅 Фаджр", "Фаджр"},
		{"☀️ Восход", "Восход"},
		{"☀️ Зухр", "Зухр"},
		{"🌤 Аср", "Аср"},
		{"🌆 Магриб", "Магриб"},
		{"🌙 Иша", "Иша"},
		{"✨ Все молитвы", "all"},
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton

	for _, p := range prayers {
		btn := tgbotapi.NewInlineKeyboardButtonData(p.Label, fmt.Sprintf("adm_prayer:%d:%s", cityID, p.Key))
		row = append(row, btn)
		if len(row) == 2 {
			rows = append(rows, row)
			row = []tgbotapi.InlineKeyboardButton{}
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}

	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к выбору города", "adm_add_rule:city:1"),
	})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	b.sendOrEditMessage(chatID, messageID, text, &keyboard)
}

// handleAdminSelectOffsetStep выбор смещения времени в минутах либо ввод фиксированного времени
func (b *Bot) handleAdminSelectOffsetStep(chatID int64, cityID int, prayer string, messageID int) {
	cityInfo, ok := api.GetCityByID(cityID)
	cityName := "Уфа"
	if ok {
		cityName = cityInfo.DisplayName
	}

	text := fmt.Sprintf(
		"➕ *Шаг 3 из 4: Задайте время молитвы*\n\n"+
			"📍 Населенный пункт: *%s*\n"+
			"🕌 Молитва: *%s*\n\n"+
			"Выберите смещение в минутах (+/-), либо *зафиксируйте точное время* (например, `13:30`):",
		cityName, prayer,
	)

	offsetsMinus := []int{-15, -10, -5, -3, -2, -1}
	offsetsPlus := []int{1, 2, 3, 5, 10, 15, 30}

	var rows [][]tgbotapi.InlineKeyboardButton

	// Кнопка для фиксации точного времени
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("⏱ Зафиксировать точное время (HH:MM)", fmt.Sprintf("adm_enter_fixed:%d:%s", cityID, prayer)),
	})

	// Кнопки убавления
	var rowMinus []tgbotapi.InlineKeyboardButton
	for _, m := range offsetsMinus {
		label := fmt.Sprintf("%d мин", m)
		rowMinus = append(rowMinus, tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("adm_offset:%d:%s:%d", cityID, prayer, m)))
		if len(rowMinus) == 3 {
			rows = append(rows, rowMinus)
			rowMinus = []tgbotapi.InlineKeyboardButton{}
		}
	}
	if len(rowMinus) > 0 {
		rows = append(rows, rowMinus)
	}

	// Кнопки прибавления
	var rowPlus []tgbotapi.InlineKeyboardButton
	for _, m := range offsetsPlus {
		label := fmt.Sprintf("+%d мин", m)
		rowPlus = append(rowPlus, tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("adm_offset:%d:%s:%d", cityID, prayer, m)))
		if len(rowPlus) == 3 {
			rows = append(rows, rowPlus)
			rowPlus = []tgbotapi.InlineKeyboardButton{}
		}
	}
	if len(rowPlus) > 0 {
		rows = append(rows, rowPlus)
	}

	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к выбору молитвы", fmt.Sprintf("adm_sel_city:%d", cityID)),
	})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	b.sendOrEditMessage(chatID, messageID, text, &keyboard)
}

// handleAdminSelectDurationStep выбор периода действия корректировки
func (b *Bot) handleAdminSelectDurationStep(chatID int64, cityID int, prayer string, offsetMinutes int, messageID int) {
	cityInfo, ok := api.GetCityByID(cityID)
	cityName := "Уфа"
	if ok {
		cityName = cityInfo.DisplayName
	}

	sign := "+"
	if offsetMinutes < 0 {
		sign = ""
	}

	text := fmt.Sprintf(
		"➕ *Шаг 4 из 4: Срок действия корректировки*\n\n"+
			"📍 Населенный пункт: *%s*\n"+
			"🕌 Молитва: *%s*\n"+
			"⏱ Смещение: *%s%d мин*\n\n"+
			"Выберите, до какого периода сохранять это правило:",
		cityName, prayer, sign, offsetMinutes,
	)

	periods := []struct {
		Label string
		Key   string
	}{
		{"🗓 7 дней", "7d"},
		{"🗓 14 дней", "14d"},
		{"🗓 До конца месяца", "endmonth"},
		{"🗓 1 месяц", "1m"},
		{"🗓 3 месяца", "3m"},
		{"🗓 1 год", "1y"},
		{"♾ Бессрочно (10 лет)", "forever"},
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton

	for _, p := range periods {
		btn := tgbotapi.NewInlineKeyboardButtonData(p.Label, fmt.Sprintf("adm_save:%d:%s:%d:%s", cityID, prayer, offsetMinutes, p.Key))
		row = append(row, btn)
		if len(row) == 2 {
			rows = append(rows, row)
			row = []tgbotapi.InlineKeyboardButton{}
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}

	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к выбору смещения", fmt.Sprintf("adm_prayer:%d:%s", cityID, prayer)),
	})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	b.sendOrEditMessage(chatID, messageID, text, &keyboard)
}

// handleAdminSelectDurationStepForFixed выбор периода действия для фиксированного времени
func (b *Bot) handleAdminSelectDurationStepForFixed(chatID int64, cityID int, prayer string, fixedTime string, messageID int) {
	cityInfo, ok := api.GetCityByID(cityID)
	cityName := "Уфа"
	if ok {
		cityName = cityInfo.DisplayName
	}

	text := fmt.Sprintf(
		"➕ *Шаг 4 из 4: Срок действия правила*\n\n"+
			"📍 Населенный пункт: *%s*\n"+
			"🕌 Молитва: *%s*\n"+
			"⏱ Фиксированное время: *%s*\n\n"+
			"Выберите, до какого периода сохранять это правило:",
		cityName, prayer, fixedTime,
	)

	periods := []struct {
		Label string
		Key   string
	}{
		{"🗓 7 дней", "7d"},
		{"🗓 14 дней", "14d"},
		{"🗓 До конца месяца", "endmonth"},
		{"🗓 1 месяц", "1m"},
		{"🗓 3 месяца", "3m"},
		{"🗓 1 год", "1y"},
		{"♾ Бессрочно (10 лет)", "forever"},
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton

	for _, p := range periods {
		btn := tgbotapi.NewInlineKeyboardButtonData(p.Label, fmt.Sprintf("adm_save_fix:%d|%s|%s|%s", cityID, prayer, fixedTime, p.Key))
		row = append(row, btn)
		if len(row) == 2 {
			rows = append(rows, row)
			row = []tgbotapi.InlineKeyboardButton{}
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}

	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к выбору времени", fmt.Sprintf("adm_prayer:%d:%s", cityID, prayer)),
	})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	b.sendOrEditMessage(chatID, messageID, text, &keyboard)
}

// handleAdminSaveRule сохраняет корректировку со смещением в БД
func (b *Bot) handleAdminSaveRule(chatID int64, cityID int, prayer string, offsetMinutes int, periodKey string, messageID int) {
	cityInfo, ok := api.GetCityByID(cityID)
	cityName := "Уфа"
	cleanCity := "Уфа"
	if ok {
		cityName = cityInfo.DisplayName
		cleanCity = cityInfo.CleanName
	}

	now := time.Now()
	var validUntil time.Time

	switch periodKey {
	case "7d":
		validUntil = now.AddDate(0, 0, 7)
	case "14d":
		validUntil = now.AddDate(0, 0, 14)
	case "endmonth":
		// Последний день текущего месяца 23:59:59
		firstOfNextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
		validUntil = firstOfNextMonth.Add(-1 * time.Second)
	case "1m":
		validUntil = now.AddDate(0, 1, 0)
	case "3m":
		validUntil = now.AddDate(0, 3, 0)
	case "1y":
		validUntil = now.AddDate(1, 0, 0)
	case "forever":
		validUntil = now.AddDate(10, 0, 0)
	default:
		validUntil = now.AddDate(0, 1, 0)
	}

	// Устанавливаем конец дня
	validUntil = time.Date(validUntil.Year(), validUntil.Month(), validUntil.Day(), 23, 59, 59, 0, validUntil.Location())

	err := b.storage.SaveAdjustment(cleanCity, prayer, offsetMinutes, "", validUntil)
	if err != nil {
		log.Printf("Ошибка сохранения корректировки: %v", err)
		b.sendOrEditMessage(chatID, messageID, "❌ Произошла ошибка при сохранении корректировки.", nil)
		return
	}

	sign := "+"
	if offsetMinutes < 0 {
		sign = ""
	}

	text := fmt.Sprintf(
		"✅ *Корректировка успешно сохранена!*\n\n"+
			"📍 *Населенный пункт:* %s\n"+
			"🕌 *Молитва:* %s\n"+
			"⏱ *Смещение:* %s%d мин\n"+
			"📅 *Действует до:* %s\n\n"+
			"Все запросы расписания и рассылки для данного города будут автоматически учитывать это правило.",
		cityName, prayer, sign, offsetMinutes, validUntil.Format("02.01.2006"),
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить еще правило", "adm_add_rule:city:1"),
			tgbotapi.NewInlineKeyboardButtonData("📋 Список правил", "adm_rules"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 В главное меню админки", "adm_main"),
		),
	)

	b.sendOrEditMessage(chatID, messageID, text, &keyboard)
}

// handleAdminSaveRuleFixed сохраняет правило с фиксированным временем в БД
func (b *Bot) handleAdminSaveRuleFixed(chatID int64, cityID int, prayer string, fixedTime string, periodKey string, messageID int) {
	cityInfo, ok := api.GetCityByID(cityID)
	cityName := "Уфа"
	cleanCity := "Уфа"
	if ok {
		cityName = cityInfo.DisplayName
		cleanCity = cityInfo.CleanName
	}

	now := time.Now()
	var validUntil time.Time

	switch periodKey {
	case "7d":
		validUntil = now.AddDate(0, 0, 7)
	case "14d":
		validUntil = now.AddDate(0, 0, 14)
	case "endmonth":
		firstOfNextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
		validUntil = firstOfNextMonth.Add(-1 * time.Second)
	case "1m":
		validUntil = now.AddDate(0, 1, 0)
	case "3m":
		validUntil = now.AddDate(0, 3, 0)
	case "1y":
		validUntil = now.AddDate(1, 0, 0)
	case "forever":
		validUntil = now.AddDate(10, 0, 0)
	default:
		validUntil = now.AddDate(0, 1, 0)
	}

	validUntil = time.Date(validUntil.Year(), validUntil.Month(), validUntil.Day(), 23, 59, 59, 0, validUntil.Location())

	err := b.storage.SaveAdjustment(cleanCity, prayer, 0, fixedTime, validUntil)
	if err != nil {
		log.Printf("Ошибка сохранения фиксированного времени: %v", err)
		b.sendOrEditMessage(chatID, messageID, "❌ Произошла ошибка при сохранении правила.", nil)
		return
	}

	text := fmt.Sprintf(
		"✅ *Фиксированное время успешно сохранено!*\n\n"+
			"📍 *Населенный пункт:* %s\n"+
			"🕌 *Молитва:* %s\n"+
			"⏱ *Фиксированное время:* %s\n"+
			"📅 *Действует до:* %s\n\n"+
			"Все запросы расписания и рассылки для данного города будут автоматически отображать указанное время.",
		cityName, prayer, fixedTime, validUntil.Format("02.01.2006"),
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить еще правило", "adm_add_rule:city:1"),
			tgbotapi.NewInlineKeyboardButtonData("📋 Список правил", "adm_rules"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 В главное меню админки", "adm_main"),
		),
	)

	b.sendOrEditMessage(chatID, messageID, text, &keyboard)
}

// handleAdminRulesList отображает список активных корректировок с возможностью удаления
func (b *Bot) handleAdminRulesList(chatID int64, messageID int) {
	rules, err := b.storage.GetAllActiveAdjustments()
	if err != nil {
		log.Printf("Ошибка получения правил: %v", err)
		b.sendOrEditMessage(chatID, messageID, "❌ Ошибка загрузки списка правил.", nil)
		return
	}

	if len(rules) == 0 {
		text := "📋 *Список активных правил*\n\nВ данный момент нет ни одной активной корректировки."
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("➕ Добавить корректировку", "adm_add_rule:city:1"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Назад в меню", "adm_main"),
			),
		)
		b.sendOrEditMessage(chatID, messageID, text, &keyboard)
		return
	}

	var sb strings.Builder
	sb.WriteString("📋 *Активные корректировки времени:*\n\n")

	var rows [][]tgbotapi.InlineKeyboardButton

	for i, r := range rules {
		if r.FixedTime != "" {
			sb.WriteString(fmt.Sprintf(
				"*%d.* 📍 *%s* | 🕌 %s: *⏱ %s (Фиксированное)*\n   📅 Действует до: _%s_\n\n",
				i+1, r.City, r.Prayer, r.FixedTime, r.ValidUntil.Format("02.01.2006"),
			))
		} else {
			sign := "+"
			if r.OffsetMinutes < 0 {
				sign = ""
			}
			sb.WriteString(fmt.Sprintf(
				"*%d.* 📍 *%s* | 🕌 %s: *%s%d мин*\n   📅 Действует до: _%s_\n\n",
				i+1, r.City, r.Prayer, sign, r.OffsetMinutes, r.ValidUntil.Format("02.01.2006"),
			))
		}

		delBtn := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("❌ Удалить #%d (%s, %s)", i+1, r.City, r.Prayer),
			fmt.Sprintf("adm_del_rule:%d", r.ID),
		)
		rows = append(rows, []tgbotapi.InlineKeyboardButton{delBtn})
	}

	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("➕ Добавить правило", "adm_add_rule:city:1"),
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "adm_main"),
	})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	b.sendOrEditMessage(chatID, messageID, sb.String(), &keyboard)
}

// handleAdminDeleteRule удаляет правило корректировки
func (b *Bot) handleAdminDeleteRule(chatID int64, ruleID int64, messageID int) {
	err := b.storage.DeleteAdjustment(ruleID)
	if err != nil {
		log.Printf("Ошибка удаления правила #%d: %v", ruleID, err)
	}
	b.handleAdminRulesList(chatID, messageID)
}

// handleAdminAdminsList отображает список администраторов и кнопки управления
func (b *Bot) handleAdminAdminsList(chatID int64, messageID int) {
	dbAdmins, err := b.storage.GetAdmins()
	if err != nil {
		log.Printf("Ошибка получения списка админов: %v", err)
	}

	var sb strings.Builder
	sb.WriteString("👥 *Администраторы бота*\n\n")

	sb.WriteString("👑 *Главные администраторы (.env):*\n")
	if len(b.configAdminIDs) == 0 {
		sb.WriteString("   _(Не заданы в .env)_\n")
	} else {
		for _, id := range b.configAdminIDs {
			sb.WriteString(fmt.Sprintf("   • ID: `%d` (Супер-админ)\n", id))
		}
	}
	sb.WriteString("\n")

	sb.WriteString("🛡 *Дополнительные администраторы (БД):*\n")
	var rows [][]tgbotapi.InlineKeyboardButton

	if len(dbAdmins) == 0 {
		sb.WriteString("   _(Нет назначенных администраторов)_\n")
	} else {
		for _, a := range dbAdmins {
			userTag := ""
			if a.Username != "" {
				userTag = " (@" + a.Username + ")"
			}
			sb.WriteString(fmt.Sprintf("   • ID: `%d`%s (добавлен: %s)\n", a.UserID, userTag, a.CreatedAt.Format("02.01.2006")))

			delBtn := tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("❌ Удалить админа %d", a.UserID),
				fmt.Sprintf("adm_del_admin:%d", a.UserID),
			)
			rows = append(rows, []tgbotapi.InlineKeyboardButton{delBtn})
		}
	}

	sb.WriteString("\n💡 _Чтобы добавить администратора, отправьте команду:_\n`/addadmin <Telegram_ID> [username]`")

	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад в меню", "adm_main"),
	})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	b.sendOrEditMessage(chatID, messageID, sb.String(), &keyboard)
}

// handleAddAdminCommand обрабатывает команду /addadmin <user_id> [username]
func (b *Bot) handleAddAdminCommand(chatID int64, fromID int64, text string) {
	if !b.storage.IsAdmin(fromID, b.configAdminIDs) {
		b.sendMessage(chatID, "⛔️ У вас нет прав администратора.")
		return
	}

	parts := strings.Fields(text)
	if len(parts) < 2 {
		b.sendMessage(chatID, "⚠️ *Использование команды:*\n`/addadmin <Telegram_User_ID> [username]`\n\nПример:\n`/addadmin 123456789 @username`")
		return
	}

	var targetID int64
	_, err := fmt.Sscanf(parts[1], "%d", &targetID)
	if err != nil || targetID == 0 {
		b.sendMessage(chatID, "❌ Некорректный ID пользователя. Укажите числовой Telegram ID.")
		return
	}

	username := ""
	if len(parts) >= 3 {
		username = strings.TrimPrefix(parts[2], "@")
	}

	err = b.storage.AddAdmin(targetID, username, fromID)
	if err != nil {
		log.Printf("Ошибка добавления администратора: %v", err)
		b.sendMessage(chatID, fmt.Sprintf("❌ Ошибка при добавлении администратора: %v", err))
		return
	}

	b.sendMessage(chatID, fmt.Sprintf("✅ Пользователь с ID `%d` успешно назначен администратором бота!", targetID))
}

// handleDelAdminCommand обрабатывает команду /deladmin <user_id>
func (b *Bot) handleDelAdminCommand(chatID int64, fromID int64, text string) {
	if !b.storage.IsAdmin(fromID, b.configAdminIDs) {
		b.sendMessage(chatID, "⛔️ У вас нет прав администратора.")
		return
	}

	parts := strings.Fields(text)
	if len(parts) < 2 {
		b.sendMessage(chatID, "⚠️ *Использование команды:*\n`/deladmin <Telegram_User_ID>`")
		return
	}

	var targetID int64
	_, err := fmt.Sscanf(parts[1], "%d", &targetID)
	if err != nil || targetID == 0 {
		b.sendMessage(chatID, "❌ Некорректный ID пользователя.")
		return
	}

	for _, cfgID := range b.configAdminIDs {
		if cfgID == targetID {
			b.sendMessage(chatID, "⛔️ Нельзя удалить главного администратора из конфигурации (.env).")
			return
		}
	}

	err = b.storage.RemoveAdmin(targetID)
	if err != nil {
		log.Printf("Ошибка удаления администратора: %v", err)
		b.sendMessage(chatID, fmt.Sprintf("❌ Ошибка при удалении: %v", err))
		return
	}

	b.sendMessage(chatID, fmt.Sprintf("✅ Администратор с ID `%d` удален.", targetID))
}

// handleAdminStats отображает статистику бота
func (b *Bot) handleAdminStats(chatID int64, messageID int) {
	totalUsers, totalGroups, totalSubscribers, activeAdjustments, totalAdmins, err := b.storage.GetStats()
	if err != nil {
		log.Printf("Ошибка получения статистики: %v", err)
	}

	text := fmt.Sprintf(
		"📊 *Статистика Namaz Time Bot*\n\n"+
			"👤 *Личные чаты пользователей:* %d\n"+
			"👥 *Telegram-группы и каналы:* %d\n"+
			"🔔 *Всего активных рассылок:* %d\n\n"+
			"🛠 *Действующих правил корректировки:* %d\n"+
			"🛡 *Администраторов в БД:* %d\n"+
			"👑 *Супер-админов (.env):* %d",
		totalUsers, totalGroups, totalSubscribers,
		activeAdjustments, totalAdmins, len(b.configAdminIDs),
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить", "adm_stats"),
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "adm_main"),
		),
	)

	b.sendOrEditMessage(chatID, messageID, text, &keyboard)
}

// sendOrEditMessage вспомогательная функция для редактирования или отправки сообщений
func (b *Bot) sendOrEditMessage(chatID int64, messageID int, text string, keyboard *tgbotapi.InlineKeyboardMarkup) {
	if messageID > 0 {
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
		editMsg.ParseMode = "Markdown"
		if keyboard != nil {
			editMsg.ReplyMarkup = keyboard
		}
		if _, err := b.api.Send(editMsg); err != nil {
			if !strings.Contains(err.Error(), "message is not modified") {
				log.Printf("Ошибка редактирования админ-сообщения: %v", err)
			}
		}
	} else {
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		if keyboard != nil {
			msg.ReplyMarkup = keyboard
		}
		if _, err := b.api.Send(msg); err != nil {
			log.Printf("Ошибка отправки админ-сообщения: %v", err)
		}
	}
}

// handleAdminCallbacks обрабатывает все callback запросы панели администратора
func (b *Bot) handleAdminCallbacks(cb *tgbotapi.CallbackQuery) bool {
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

	if !strings.HasPrefix(data, "adm_") {
		return false
	}

	if !b.storage.IsAdmin(fromID, b.configAdminIDs) {
		b.answerCallback(cb.ID, "⛔️ Доступ запрещен.")
		return true
	}

	switch {
	case data == "adm_main":
		b.handleAdmin(chatID, fromID, messageID)
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "adm_channels:"):
		pageStr := strings.TrimPrefix(data, "adm_channels:")
		page, _ := strconv.Atoi(pageStr)
		b.handleMyChannelsList(chatID, fromID, page, messageID)
		b.answerCallback(cb.ID, "")

	case data == "adm_add_channel":
		helpText := "📢 *Как подключить Telegram-канал или группу:*\n\n" +
			"1. Перейдите в ваш канал или группу в Telegram.\n" +
			"2. Добавьте этого бота в администраторы (для канала обязательно право *«Публикация сообщений»*).\n" +
			"3. Отправьте команду в этот чат:\n" +
			"   `/channel @username_канала`\n\n" +
			"💡 *Либо просто перешлите сюда любой пост из вашего канала!*"
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 К списку каналов", "adm_channels:1"),
			),
		)
		b.sendOrEditMessage(chatID, messageID, helpText, &keyboard)
		b.answerCallback(cb.ID, "")

	case data == "adm_rules":
		b.handleAdminRulesList(chatID, messageID)
		b.answerCallback(cb.ID, "")

	case data == "adm_admins":
		b.handleAdminAdminsList(chatID, messageID)
		b.answerCallback(cb.ID, "")

	case data == "adm_stats":
		b.handleAdminStats(chatID, messageID)
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "adm_add_rule:city:"):
		pageStr := strings.TrimPrefix(data, "adm_add_rule:city:")
		page, _ := strconv.Atoi(pageStr)
		b.handleAdminAddRuleCityStep(chatID, true, page, messageID)
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "adm_add_rule:district:"):
		pageStr := strings.TrimPrefix(data, "adm_add_rule:district:")
		page, _ := strconv.Atoi(pageStr)
		b.handleAdminAddRuleCityStep(chatID, false, page, messageID)
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "adm_sel_city:"):
		cityIDStr := strings.TrimPrefix(data, "adm_sel_city:")
		cityID, _ := strconv.Atoi(cityIDStr)
		b.handleAdminSelectPrayerStep(chatID, cityID, messageID)
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "adm_prayer:"):
		parts := strings.Split(strings.TrimPrefix(data, "adm_prayer:"), ":")
		if len(parts) >= 2 {
			cityID, _ := strconv.Atoi(parts[0])
			prayer := parts[1]
			b.handleAdminSelectOffsetStep(chatID, cityID, prayer, messageID)
		}
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "adm_enter_fixed:"):
		parts := strings.Split(strings.TrimPrefix(data, "adm_enter_fixed:"), ":")
		if len(parts) >= 2 {
			cityID, _ := strconv.Atoi(parts[0])
			prayer := parts[1]
			b.setUserState(fromID, userState{
				action:    pendingActionFixedTime,
				cityID:    cityID,
				prayerKey: prayer,
			})
			cityInfo, _ := api.GetCityByID(cityID)
			cityName := "Уфа"
			if cityInfo.DisplayName != "" {
				cityName = cityInfo.DisplayName
			}
			prompt := fmt.Sprintf(
				"⏱ *Введите фиксированное время для «%s» (%s)*\n\n"+
					"Введите точное время в формате `ЧЧ:ММ` (например: `13:30` или `06:00`).\n\n"+
					"В расписании эта молитва будет всегда отображаться с этим временем.\n\n"+
					"Отправьте время ответным сообщением либо `-` для отмены.",
				escapeMarkdown(prayer), escapeMarkdown(cityName),
			)
			b.sendMessage(chatID, prompt)
		}
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "adm_save_fix:"):
		parts := strings.Split(strings.TrimPrefix(data, "adm_save_fix:"), "|")
		if len(parts) >= 4 {
			cityID, _ := strconv.Atoi(parts[0])
			prayer := parts[1]
			fixedTime := parts[2]
			periodKey := parts[3]
			b.handleAdminSaveRuleFixed(chatID, cityID, prayer, fixedTime, periodKey, messageID)
		}
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "adm_offset:"):
		parts := strings.Split(strings.TrimPrefix(data, "adm_offset:"), ":")
		if len(parts) >= 3 {
			cityID, _ := strconv.Atoi(parts[0])
			prayer := parts[1]
			offsetMinutes, _ := strconv.Atoi(parts[2])
			b.handleAdminSelectDurationStep(chatID, cityID, prayer, offsetMinutes, messageID)
		}
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "adm_save:"):
		parts := strings.Split(strings.TrimPrefix(data, "adm_save:"), ":")
		if len(parts) >= 4 {
			cityID, _ := strconv.Atoi(parts[0])
			prayer := parts[1]
			offsetMinutes, _ := strconv.Atoi(parts[2])
			periodKey := parts[3]
			b.handleAdminSaveRule(chatID, cityID, prayer, offsetMinutes, periodKey, messageID)
		}
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "adm_del_rule:"):
		ruleIDStr := strings.TrimPrefix(data, "adm_del_rule:")
		ruleID, _ := strconv.ParseInt(ruleIDStr, 10, 64)
		b.handleAdminDeleteRule(chatID, ruleID, messageID)
		b.answerCallback(cb.ID, "Правило удалено")

	case strings.HasPrefix(data, "adm_del_admin:"):
		adminIDStr := strings.TrimPrefix(data, "adm_del_admin:")
		adminID, _ := strconv.ParseInt(adminIDStr, 10, 64)
		_ = b.storage.RemoveAdmin(adminID)
		b.handleAdminAdminsList(chatID, messageID)
		b.answerCallback(cb.ID, "Администратор удален")
	}

	return true
}
