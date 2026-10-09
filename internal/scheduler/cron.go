package scheduler

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/robfig/cron/v3"

	"namaz-time-bot/internal/api"
	"namaz-time-bot/internal/bot"
	"namaz-time-bot/internal/storage"
)

type Scheduler struct {
	cron    *cron.Cron
	bot     *bot.Bot
	client  *api.Client
	storage *storage.Storage
	city    string
	loc     *time.Location
}

func New(bot *bot.Bot, client *api.Client, store *storage.Storage, city string) (*Scheduler, error) {
	location, err := time.LoadLocation("Asia/Yekaterinburg")
	if err != nil {
		location = time.FixedZone("UTC+5", 5*3600)
	}

	c := cron.New(cron.WithLocation(location))

	s := &Scheduler{
		cron:    c,
		bot:     bot,
		client:  client,
		storage: store,
		city:    city,
		loc:     location,
	}

	// Запуск каждые 1 минуту для точной проверки времени
	_, err = c.AddFunc("* * * * *", s.checkAndSendReminders)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Scheduler) Start() {
	s.cron.Start()
	log.Println("⏰ Минутный планировщик напоминаний запущен")
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}

func (s *Scheduler) checkAndSendReminders() {
	now := time.Now().In(s.loc)
	currentTimeStr := now.Format("15:04") // Формат "HH:MM", например "06:00" или "13:30"

	subscribers, err := s.storage.GetSubscribers()
	if err != nil || len(subscribers) == 0 {
		return
	}

	// 1. Проверка ежедневной рассылки хадиса дня
	s.checkAndSendDailyHadith(now, currentTimeStr, subscribers)

	// 2. Проверка напоминаний о посте (Пн/Чт и Белые дни 13, 14, 15)
	s.checkAndSendFastingHadith(now, currentTimeStr, subscribers)

	// 3. Проверка хадисов о наступающих лунных месяцах (за 1-2 дня до месяца)
	s.checkAndSendLunarMonthHadith(now, currentTimeStr, subscribers)

	// 4. Проверка ежедневной рассылки прекрасных имён Аллаха
	s.checkAndSendDailyAsma(now, currentTimeStr, subscribers)

	cityCache := make(map[string]*api.DUMRBItem)
	tomorrowCache := make(map[string]*api.DUMRBItem)

	for _, user := range subscribers {
		userCity := user.City
		if userCity == "" {
			userCity = s.city
		}

		timing, exists := cityCache[userCity]
		if !exists {
			t, err := s.client.FetchPrayerTimes(userCity, now)
			if err != nil {
				log.Printf("Ошибка получения времени намаза для %s в планировщике: %v", userCity, err)
				continue
			}
			cityCache[userCity] = t
			timing = t
		}

		prayers := map[string]string{
			"Фаджр":  timing.Fajr,
			"Зухр":   timing.Dhuhr,
			"Аср":    timing.Asr,
			"Магриб": timing.Maghrib,
			"Иша":    timing.Isha,
		}

		cfg := api.PrayerFormatConfig{
			Preset:        user.PrayerNamesPreset,
			CustomHeader:  user.CustomHeader,
			CustomFooter:  user.CustomFooter,
			CustomPrayers: storage.ParseCustomPrayerNames(user.CustomPrayerNames),
		}

		msgDaily := api.FormatMessageCustom(timing, userCity, now, cfg)

		// 1. Рассылка расписания по всем настроенным временам
		broadcastTimes := user.GetParsedBroadcastTimes()
		for _, bTime := range broadcastTimes {
			if bTime == currentTimeStr {
				if now.Hour() >= 16 {
					tomorrowTiming, tExists := tomorrowCache[userCity]
					if !tExists {
						tomorrowDate := now.AddDate(0, 0, 1)
						tt, err := s.client.FetchPrayerTimes(userCity, tomorrowDate)
						if err != nil {
							log.Printf("Ошибка получения завтрашнего расписания для %s: %v", userCity, err)
						} else {
							tomorrowCache[userCity] = tt
							tomorrowTiming = tt
						}
					}
					if tomorrowTiming != nil {
						msgEvening := api.FormatEveningMessageCustom(timing, tomorrowTiming, userCity, now, cfg)
						s.bot.SendToChatWithShare(user.ChatID, msgEvening)
					} else {
						s.bot.SendToChatWithShare(user.ChatID, msgDaily)
					}
				} else {
					s.bot.SendToChatWithShare(user.ChatID, msgDaily)
				}
			}
		}

		// 2. Напоминания о намазах
		for name, timeStr := range prayers {
			// В момент наступления
			if user.NotifyAtTime && timeStr == currentTimeStr {
				msg := fmt.Sprintf("🕌 *Наступило время намаза %s в г. %s!* (%s)", name, userCity, timeStr)

				// Добавление хадиса о соответствующей молитве
				if user.HadithPrayerEnabled && s.storage.GetSetting("hadith_prayer_enabled", "1") == "1" {
					cat := getPrayerHadithCategory(name, now.Weekday())
					if h, err := s.storage.GetRandomHadith(cat); err == nil && h != nil {
						msg += fmt.Sprintf("\n\n📖 *Хадис:*\n«%s»\n📚 _%s_", h.Text, h.Source)
					}
				}

				s.bot.SendToChat(user.ChatID, msg)
			}

			// За 15 минут до намаза
			if user.Notify15Min {
				fifteenMinBefore := subtractMinutes(timeStr, 15)
				if fifteenMinBefore == currentTimeStr {
					msg := fmt.Sprintf("⏳ *До намаза %s осталось 15 минут!* (%s, г. %s)", name, timeStr, userCity)
					s.bot.SendToChat(user.ChatID, msg)
				}
			}
		}

		time.Sleep(20 * time.Millisecond)
	}
}

// checkAndSendDailyHadith отправляет ежедневный хадис дня на случайную тему
func (s *Scheduler) checkAndSendDailyHadith(now time.Time, currentTimeStr string, subscribers []storage.User) {
	if s.storage.GetSetting("hadith_daily_enabled", "1") != "1" {
		return
	}
	targetTime := s.storage.GetSetting("hadith_daily_time", "09:00")
	if currentTimeStr != targetTime {
		return
	}

	todayStr := now.Format("2006-01-02")
	if s.storage.GetSetting("last_daily_hadith_date", "") == todayStr {
		return
	}
	_ = s.storage.SetSetting("last_daily_hadith_date", todayStr)

	h, err := s.storage.GetRandomHadith(storage.CategoryGeneral)
	if err != nil || h == nil {
		h, err = s.storage.GetRandomHadithAny()
		if err != nil || h == nil {
			return
		}
	}

	msg := fmt.Sprintf("📖 *Хадис дня*\n\n«%s»\n\n📚 *Источник:* %s", h.Text, h.Source)
	for _, user := range subscribers {
		if user.HadithDailyEnabled {
			s.bot.SendToChatWithShare(user.ChatID, msg)
			time.Sleep(20 * time.Millisecond)
		}
	}
}

// checkAndSendDailyAsma отправляет ежедневное имя Аллаха
func (s *Scheduler) checkAndSendDailyAsma(now time.Time, currentTimeStr string, subscribers []storage.User) {
	if s.storage.GetSetting("asma_daily_enabled", "1") != "1" {
		return
	}
	targetTime := s.storage.GetSetting("asma_daily_time", "10:00")
	if currentTimeStr != targetTime {
		return
	}

	todayStr := now.Format("2006-01-02")
	if s.storage.GetSetting("last_daily_asma_date", "") == todayStr {
		return
	}
	_ = s.storage.SetSetting("last_daily_asma_date", todayStr)

	lastID, _ := strconv.Atoi(s.storage.GetSetting("last_asma_id", "0"))
	nextID := (lastID % 100) + 1
	_ = s.storage.SetSetting("last_asma_id", strconv.Itoa(nextID))

	item, err := storage.GetAsmaName(nextID)
	if err != nil || item == nil {
		return
	}

	for _, user := range subscribers {
		if user.AsmaDailyEnabled {
			s.bot.SendDailyAsma(user.ChatID, item)
			time.Sleep(20 * time.Millisecond)
		}
	}
}

// checkAndSendFastingHadith проверяет дни поста (Пн/Чт и Белые дни 13, 14, 15) и отправляет хадис
func (s *Scheduler) checkAndSendFastingHadith(now time.Time, currentTimeStr string, subscribers []storage.User) {
	if s.storage.GetSetting("hadith_fasting_enabled", "1") != "1" {
		return
	}
	targetTime := s.storage.GetSetting("hadith_fasting_time", "07:00")
	if currentTimeStr != targetTime {
		return
	}

	todayStr := now.Format("2006-01-02")
	if s.storage.GetSetting("last_fasting_hadith_date", "") == todayStr {
		return
	}

	hd, err := api.GetHijriDate(now, api.GetGlobalHijriOffset())
	if err == nil && (hd.Day == 13 || hd.Day == 14 || hd.Day == 15) {
		_ = s.storage.SetSetting("last_fasting_hadith_date", todayStr)
		h, _ := s.storage.GetRandomHadith(storage.CategoryFastingWhiteDays)
		if h == nil {
			h, _ = s.storage.GetRandomHadith(storage.CategoryFastingGeneral)
		}
		if h != nil {
			monthName := api.GetHijriMonthName(hd.Month, "ru")
			msg := fmt.Sprintf(
				"🌕 *Сунна поста: Белые дни (%s)*\n\n"+
					"Сегодня *%d-е число месяца %s* — один из трех дней (13, 14, 15 числа по лунному календарю), когда сунной является соблюдать пост.\n\n"+
					"«%s»\n\n"+
					"📚 *Источник:* %s",
				monthName, hd.Day, monthName, h.Text, h.Source,
			)
			for _, user := range subscribers {
				if user.HadithFastingEnabled {
					s.bot.SendToChatWithShare(user.ChatID, msg)
					time.Sleep(20 * time.Millisecond)
				}
			}
		}
		return
	}

	if now.Weekday() == time.Monday || now.Weekday() == time.Thursday {
		_ = s.storage.SetSetting("last_fasting_hadith_date", todayStr)
		dayName := "Понедельник"
		if now.Weekday() == time.Thursday {
			dayName = "Четверг"
		}
		h, _ := s.storage.GetRandomHadith(storage.CategoryFastingMonThu)
		if h == nil {
			h, _ = s.storage.GetRandomHadith(storage.CategoryFastingGeneral)
		}
		if h != nil {
			msg := fmt.Sprintf(
				"📅 *Сунна поста: %s*\n\n"+
					"Сегодня *%s* — благословенный день, в который является сунной соблюдать желательный пост.\n\n"+
					"«%s»\n\n"+
					"📚 *Источник:* %s",
				dayName, dayName, h.Text, h.Source,
			)
			for _, user := range subscribers {
				if user.HadithFastingEnabled {
					s.bot.SendToChatWithShare(user.ChatID, msg)
					time.Sleep(20 * time.Millisecond)
				}
			}
		}
	}
}

// checkAndSendLunarMonthHadith отправляет хадис о наступающем лунном месяце за 1-2 дня до его начала
func (s *Scheduler) checkAndSendLunarMonthHadith(now time.Time, currentTimeStr string, subscribers []storage.User) {
	if s.storage.GetSetting("hadith_months_enabled", "1") != "1" {
		return
	}
	targetTime := s.storage.GetSetting("hadith_months_time", "12:00")
	if currentTimeStr != targetTime {
		return
	}

	hd, err := api.GetHijriDate(now, api.GetGlobalHijriOffset())
	if err != nil || hd.Day < 29 {
		return
	}

	nextMonth := (hd.Month % 12) + 1
	nextYear := hd.Year
	if hd.Month == 12 {
		nextYear++
	}

	transitionKey := fmt.Sprintf("%d-%02d", nextYear, nextMonth)
	if s.storage.GetSetting("last_month_hadith_key", "") == transitionKey {
		return
	}

	var cat string
	switch nextMonth {
	case 1:
		cat = storage.CategoryMonthMuharram
	case 7:
		cat = storage.CategoryMonthRajab
	case 8:
		cat = storage.CategoryMonthShaban
	case 9:
		cat = storage.CategoryMonthRamadan
	case 10:
		cat = storage.CategoryMonthShawwal
	case 11:
		cat = storage.CategoryMonthDhulQadah
	case 12:
		cat = storage.CategoryMonthDhulHijjah
	}

	if cat == "" {
		return
	}

	h, _ := s.storage.GetRandomHadith(cat)
	if h == nil {
		return
	}

	_ = s.storage.SetSetting("last_month_hadith_key", transitionKey)
	monthName := api.GetHijriMonthName(nextMonth, "ru")
	msg := fmt.Sprintf(
		"🌙 *Приближается благословенный месяц %s!*\n\n"+
			"«%s»\n\n"+
			"📚 *Источник:* %s",
		monthName, h.Text, h.Source,
	)

	for _, user := range subscribers {
		if user.HadithDailyEnabled {
			s.bot.SendToChatWithShare(user.ChatID, msg)
			time.Sleep(20 * time.Millisecond)
		}
	}
}

func getPrayerHadithCategory(name string, wd time.Weekday) string {
	switch name {
	case "Фаджр":
		return storage.CategoryPrayerFajr
	case "Зухр":
		if wd == time.Friday {
			return storage.CategoryPrayerJumah
		}
		return storage.CategoryPrayerDhuhr
	case "Аср":
		return storage.CategoryPrayerAsr
	case "Магриб":
		return storage.CategoryPrayerMaghrib
	case "Иша":
		return storage.CategoryPrayerIsha
	default:
		return storage.CategoryGeneral
	}
}

// subtractMinutes вычитает минуты из времени формата "15:04"
func subtractMinutes(timeStr string, mins int) string {
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return ""
	}
	return t.Add(-time.Duration(mins) * time.Minute).Format("15:04")
}
