package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type User struct {
	ChatID            int64
	City              string
	SubscribedAt      time.Time
	DailyScheduleTime string
	Notify15Min       bool
	NotifyAtTime      bool
}

type BotAdmin struct {
	UserID    int64
	Username  string
	AddedBy   int64
	CreatedAt time.Time
}

type PrayerAdjustment struct {
	ID            int64
	City          string
	Prayer        string // "Фаджр", "Восход", "Зухр", "Аср", "Магриб", "Иша" или "all"
	OffsetMinutes int
	ValidUntil    time.Time
	CreatedAt     time.Time
}

type Storage struct {
	db *sql.DB
}

func New(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия БД: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка соединения с БД: %w", err)
	}

	s := &Storage{db: db}
	if err := s.init(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Storage) init() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			chat_id INTEGER PRIMARY KEY,
			city TEXT NOT NULL,
			subscribed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			daily_schedule_time TEXT DEFAULT '06:00',
			notify_15_min BOOLEAN DEFAULT TRUE,
			notify_at_time BOOLEAN DEFAULT TRUE
		);`,
		`CREATE TABLE IF NOT EXISTS bot_admins (
			user_id INTEGER PRIMARY KEY,
			username TEXT DEFAULT '',
			added_by INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS prayer_adjustments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			city TEXT NOT NULL,
			prayer TEXT NOT NULL,
			offset_minutes INTEGER NOT NULL,
			valid_until DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
	}

	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("ошибка инициализации БД (%s): %w", q, err)
		}
	}

	_ = s.addColumnIfNotExist("daily_schedule_time", "TEXT DEFAULT '06:00'")
	_ = s.addColumnIfNotExist("notify_15min", "BOOLEAN DEFAULT 1")
	_ = s.addColumnIfNotExist("notify_at_time", "BOOLEAN DEFAULT 1")
	return nil
}

func (s *Storage) addColumnIfNotExist(column, colType string) error {
	query := fmt.Sprintf("ALTER TABLE users ADD COLUMN %s %s;", column, colType)
	_, err := s.db.Exec(query)
	return err // Игнорируем ошибку, если колонка уже существует
}

// --- Управление пользователями и чатами ---

func (s *Storage) GetUser(chatID int64) (*User, error) {
	query := `SELECT chat_id, city, subscribed_at, daily_schedule_time, notify_15min, notify_at_time FROM users WHERE chat_id = ?;`
	row := s.db.QueryRow(query, chatID)

	var u User
	err := row.Scan(&u.ChatID, &u.City, &u.SubscribedAt, &u.DailyScheduleTime, &u.Notify15Min, &u.NotifyAtTime)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Storage) SetUserCity(chatID int64, city string) error {
	query := `
	INSERT INTO users (chat_id, city, subscribed_at, daily_schedule_time, notify_15min, notify_at_time)
	VALUES (?, ?, ?, '06:00', 1, 1)
	ON CONFLICT(chat_id) DO UPDATE SET city = excluded.city;
	`
	_, err := s.db.Exec(query, chatID, city, time.Now())
	return err
}

func (s *Storage) GetUserCity(chatID int64, defaultCity string) (string, error) {
	query := `SELECT city FROM users WHERE chat_id = ?;`
	var city string
	err := s.db.QueryRow(query, chatID).Scan(&city)
	if err == sql.ErrNoRows || city == "" {
		return defaultCity, nil
	}
	if err != nil {
		return defaultCity, err
	}
	return city, nil
}

func (s *Storage) UpdateDailyTime(chatID int64, dailyTime string) error {
	query := `
	INSERT INTO users (chat_id, city, subscribed_at, daily_schedule_time, notify_15min, notify_at_time)
	VALUES (?, 'Уфа', ?, ?, 1, 1)
	ON CONFLICT(chat_id) DO UPDATE SET daily_schedule_time = excluded.daily_schedule_time;
	`
	_, err := s.db.Exec(query, chatID, time.Now(), dailyTime)
	return err
}

func (s *Storage) ToggleNotify15min(chatID int64) error {
	query := `
	INSERT INTO users (chat_id, city, subscribed_at, daily_schedule_time, notify_15min, notify_at_time)
	VALUES (?, 'Уфа', ?, '06:00', 0, 1)
	ON CONFLICT(chat_id) DO UPDATE SET notify_15min = CASE WHEN notify_15min = 1 THEN 0 ELSE 1 END;
	`
	_, err := s.db.Exec(query, chatID, time.Now())
	return err
}

func (s *Storage) ToggleNotifyAtTime(chatID int64) error {
	query := `
	INSERT INTO users (chat_id, city, subscribed_at, daily_schedule_time, notify_15min, notify_at_time)
	VALUES (?, 'Уфа', ?, '06:00', 1, 0)
	ON CONFLICT(chat_id) DO UPDATE SET notify_at_time = CASE WHEN notify_at_time = 1 THEN 0 ELSE 1 END;
	`
	_, err := s.db.Exec(query, chatID, time.Now())
	return err
}

func (s *Storage) Subscribe(chatID int64, city string) error {
	query := `
	INSERT INTO users (chat_id, city, subscribed_at)
	VALUES (?, ?, ?)
	ON CONFLICT(chat_id) DO UPDATE SET city = excluded.city;
	`
	_, err := s.db.Exec(query, chatID, city, time.Now())
	if err != nil {
		return fmt.Errorf("ошибка добавления пользователя: %w", err)
	}
	return nil
}

func (s *Storage) Unsubscribe(chatID int64) error {
	query := `DELETE FROM users WHERE chat_id = ?;`
	_, err := s.db.Exec(query, chatID)
	if err != nil {
		return fmt.Errorf("ошибка удаления пользователя: %w", err)
	}
	return nil
}

func (s *Storage) GetSubscribers() ([]User, error) {
	query := `SELECT chat_id, city, subscribed_at, daily_schedule_time, notify_15min, notify_at_time FROM users;`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения подписчиков: %w", err)
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ChatID, &u.City, &u.SubscribedAt, &u.DailyScheduleTime, &u.Notify15Min, &u.NotifyAtTime); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// --- Управление администраторами ---

func (s *Storage) IsAdmin(userID int64, configAdmins []int64) bool {
	for _, id := range configAdmins {
		if id == userID {
			return true
		}
	}

	query := `SELECT 1 FROM bot_admins WHERE user_id = ? LIMIT 1;`
	var exists int
	err := s.db.QueryRow(query, userID).Scan(&exists)
	return err == nil && exists == 1
}

func (s *Storage) AddAdmin(userID int64, username string, addedBy int64) error {
	query := `
	INSERT INTO bot_admins (user_id, username, added_by, created_at)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(user_id) DO UPDATE SET username = excluded.username;
	`
	_, err := s.db.Exec(query, userID, username, addedBy, time.Now())
	return err
}

func (s *Storage) RemoveAdmin(userID int64) error {
	query := `DELETE FROM bot_admins WHERE user_id = ?;`
	_, err := s.db.Exec(query, userID)
	return err
}

func (s *Storage) GetAdmins() ([]BotAdmin, error) {
	query := `SELECT user_id, username, added_by, created_at FROM bot_admins ORDER BY created_at ASC;`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []BotAdmin
	for rows.Next() {
		var a BotAdmin
		if err := rows.Scan(&a.UserID, &a.Username, &a.AddedBy, &a.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// --- Корректировки времени намаза ---

func (s *Storage) SaveAdjustment(city, prayer string, offsetMinutes int, validUntil time.Time) error {
	// Сначала удаляем предыдущие правила для того же города и молитвы
	_, _ = s.db.Exec(`DELETE FROM prayer_adjustments WHERE city = ? AND prayer = ?;`, city, prayer)

	query := `
	INSERT INTO prayer_adjustments (city, prayer, offset_minutes, valid_until, created_at)
	VALUES (?, ?, ?, ?, ?);
	`
	_, err := s.db.Exec(query, city, prayer, offsetMinutes, validUntil, time.Now())
	return err
}

func (s *Storage) GetActiveAdjustments(city string, date time.Time) (map[string]int, error) {
	query := `
	SELECT prayer, offset_minutes FROM prayer_adjustments 
	WHERE city = ? AND valid_until >= ?;
	`
	// Сравниваем с началом текущего дня даты
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	rows, err := s.db.Query(query, city, dayStart)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	adjustments := make(map[string]int)
	for rows.Next() {
		var prayer string
		var offset int
		if err := rows.Scan(&prayer, &offset); err != nil {
			return nil, err
		}
		adjustments[prayer] = offset
	}
	return adjustments, rows.Err()
}

func (s *Storage) GetAllActiveAdjustments() ([]PrayerAdjustment, error) {
	query := `
	SELECT id, city, prayer, offset_minutes, valid_until, created_at 
	FROM prayer_adjustments 
	WHERE valid_until >= ?
	ORDER BY city ASC, id ASC;
	`
	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	rows, err := s.db.Query(query, dayStart)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []PrayerAdjustment
	for rows.Next() {
		var a PrayerAdjustment
		if err := rows.Scan(&a.ID, &a.City, &a.Prayer, &a.OffsetMinutes, &a.ValidUntil, &a.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

func (s *Storage) DeleteAdjustment(id int64) error {
	query := `DELETE FROM prayer_adjustments WHERE id = ?;`
	_, err := s.db.Exec(query, id)
	return err
}

// --- Статистика бота ---

func (s *Storage) GetStats() (totalUsers, totalGroups, totalSubscribers, activeAdjustments, totalAdmins int, err error) {
	// 1. Пользователи и группы
	rows, err := s.db.Query(`SELECT chat_id FROM users;`)
	if err != nil {
		return 0, 0, 0, 0, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var chatID int64
		if err := rows.Scan(&chatID); err == nil {
			totalSubscribers++
			if chatID < 0 {
				totalGroups++
			} else {
				totalUsers++
			}
		}
	}
	if err := rows.Err(); err != nil {
		return 0, 0, 0, 0, 0, err
	}

	// 2. Активные корректировки
	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM prayer_adjustments WHERE valid_until >= ?;`, dayStart).Scan(&activeAdjustments)

	// 3. Администраторы в БД
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM bot_admins;`).Scan(&totalAdmins)

	return totalUsers, totalGroups, totalSubscribers, activeAdjustments, totalAdmins, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}


