package scheduler

import (
	"fmt"
	"log"
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

		// 3. Напоминания о намазах
		for name, timeStr := range prayers {
			// В момент наступления
			if user.NotifyAtTime && timeStr == currentTimeStr {
				msg := fmt.Sprintf("🕌 *Наступило время намаза %s в г. %s!* (%s)", name, userCity, timeStr)
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

// subtractMinutes вычитает минуты из времени формата "15:04"
func subtractMinutes(timeStr string, mins int) string {
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return ""
	}
	return t.Add(-time.Duration(mins) * time.Minute).Format("15:04")
}
