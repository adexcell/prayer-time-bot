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
	query := `
	CREATE TABLE IF NOT EXISTS users (
		chat_id INTEGER PRIMARY KEY,
		city TEXT NOT NULL,
		subscribed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		daily_schedule_time TEXT DEFAULT '06:00',
		notify_15_min BOOLEAN DEFAULT TRUE,
		notify_at_time BOOLEAN DEFAULT TRUE
	);`

	_, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы users: %w", err)
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
	// Сначала проверяем, есть ли пользователь, если нет — создаем с городом по умолчанию
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

func (s *Storage) Close() error {
	return s.db.Close()
}

