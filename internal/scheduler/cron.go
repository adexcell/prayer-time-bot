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

		msgDaily := api.FormatMessage(timing, userCity, now)

		// 1. Рассылка дневного расписания в выбранное пользователем время
		dailyTime := user.DailyScheduleTime
		if dailyTime == "" {
			dailyTime = "06:00"
		}
		if dailyTime == currentTimeStr {
			s.bot.SendToChat(user.ChatID, msgDaily)
		}

		// 2. Напоминания о намазах
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
					msg := fmt.Sprintf("⏳ *До намаза %s осталась 15 минут!* (%s, г. %s)", name, timeStr, userCity)
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
