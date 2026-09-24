package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"namaz-time-bot/internal/api"
)

func normalizeScheduleTime(t string) string {
	t = strings.TrimSpace(t)
	if t == "" || t == "off" {
		return ""
	}
	if len(t) == 1 || len(t) == 2 {
		if val, err := strconv.Atoi(t); err == nil && val >= 0 && val <= 23 {
			return fmt.Sprintf("%02d:00", val)
		}
	}
	return t
}

// NormalizeBroadcastTime валидирует и нормализует строку времени в формат "HH:MM"
func NormalizeBroadcastTime(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("время не может быть пустым")
	}
	input = strings.ReplaceAll(input, ".", ":")
	input = strings.ReplaceAll(input, "-", ":")
	input = strings.ReplaceAll(input, " ", ":")

	parts := strings.Split(input, ":")
	var h, m int
	var err error

	if len(parts) == 1 {
		h, err = strconv.Atoi(parts[0])
		if err != nil || h < 0 || h > 23 {
			return "", fmt.Errorf("некорректный час (0-23): %s", parts[0])
		}
		m = 0
	} else if len(parts) == 2 {
		h, err = strconv.Atoi(parts[0])
		if err != nil || h < 0 || h > 23 {
			return "", fmt.Errorf("некорректный час (0-23): %s", parts[0])
		}
		m, err = strconv.Atoi(parts[1])
		if err != nil || m < 0 || m > 59 {
			return "", fmt.Errorf("некорректные минуты (0-59): %s", parts[1])
		}
	} else {
		return "", fmt.Errorf("неверный формат времени, ожидается ЧЧ:ММ")
	}

	return fmt.Sprintf("%02d:%02d", h, m), nil
}

func ParseBroadcastTimes(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}

	if strings.HasPrefix(raw, "[") {
		var list []string
		if err := json.Unmarshal([]byte(raw), &list); err == nil {
			var cleaned []string
			seen := make(map[string]bool)
			for _, t := range list {
				if norm, err := NormalizeBroadcastTime(t); err == nil && !seen[norm] {
					seen[norm] = true
					cleaned = append(cleaned, norm)
				}
			}
			sort.Strings(cleaned)
			return cleaned
		}
	}

	raw = strings.ReplaceAll(raw, ";", ",")
	raw = strings.ReplaceAll(raw, "\n", ",")
	tokens := strings.Split(raw, ",")
	seen := make(map[string]bool)
	var res []string
	for _, tok := range tokens {
		norm, err := NormalizeBroadcastTime(tok)
		if err == nil && !seen[norm] {
			seen[norm] = true
			res = append(res, norm)
		}
	}
	sort.Strings(res)
	return res
}

func ParseCustomPrayerNames(rawJSON string) map[string]string {
	res := make(map[string]string)
	if strings.TrimSpace(rawJSON) == "" {
		return res
	}
	_ = json.Unmarshal([]byte(rawJSON), &res)
	return res
}

type User struct {
	ChatID              int64
	City                string
	SubscribedAt        time.Time
	DailyScheduleTime   string
	EveningScheduleTime string
	BroadcastTimes      string
	Notify15Min         bool
	NotifyAtTime        bool
	Title               string
	ChatType            string
	AddedBy             int64
	CustomFooter        string
	CustomHeader        string
	PrayerNamesPreset   string
	CustomPrayerNames   string
}

func (u *User) GetParsedBroadcastTimes() []string {
	if u == nil {
		return []string{}
	}
	times := ParseBroadcastTimes(u.BroadcastTimes)
	if len(times) == 0 && u.BroadcastTimes == "" {
		if u.DailyScheduleTime != "" && u.DailyScheduleTime != "off" {
			if norm, err := NormalizeBroadcastTime(u.DailyScheduleTime); err == nil {
				times = append(times, norm)
			}
		}
		if u.EveningScheduleTime != "" && u.EveningScheduleTime != "off" {
			if norm, err := NormalizeBroadcastTime(u.EveningScheduleTime); err == nil {
				times = append(times, norm)
			}
		}
		sort.Strings(times)
	}
	return times
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
	FixedTime     string // например "13:30"
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
			evening_schedule_time TEXT DEFAULT '',
			broadcast_times TEXT DEFAULT '["06:00"]',
			notify_15_min BOOLEAN DEFAULT TRUE,
			notify_at_time BOOLEAN DEFAULT TRUE,
			title TEXT DEFAULT '',
			chat_type TEXT DEFAULT 'private',
			added_by INTEGER DEFAULT 0,
			custom_footer TEXT DEFAULT '',
			custom_header TEXT DEFAULT '',
			prayer_names_preset TEXT DEFAULT 'ru',
			custom_prayer_names TEXT DEFAULT ''
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
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			fixed_time TEXT DEFAULT ''
		);`,
	}

	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("ошибка инициализации БД (%s): %w", q, err)
		}
	}

	_ = s.addColumnIfNotExist("daily_schedule_time", "TEXT DEFAULT '06:00'")
	_ = s.addColumnIfNotExist("evening_schedule_time", "TEXT DEFAULT ''")
	_ = s.addColumnIfNotExist("broadcast_times", "TEXT DEFAULT '[\"06:00\"]'")
	_ = s.addColumnIfNotExist("notify_15min", "BOOLEAN DEFAULT 1")
	_ = s.addColumnIfNotExist("notify_at_time", "BOOLEAN DEFAULT 1")
	_ = s.addColumnIfNotExist("title", "TEXT DEFAULT ''")
	_ = s.addColumnIfNotExist("chat_type", "TEXT DEFAULT 'private'")
	_ = s.addColumnIfNotExist("added_by", "INTEGER DEFAULT 0")
	_ = s.addColumnIfNotExist("custom_footer", "TEXT DEFAULT ''")
	_ = s.addColumnIfNotExist("custom_header", "TEXT DEFAULT ''")
	_ = s.addColumnIfNotExist("prayer_names_preset", "TEXT DEFAULT 'ru'")
	_ = s.addColumnIfNotExist("custom_prayer_names", "TEXT DEFAULT ''")
	_ = s.addTableColumnIfNotExist("prayer_adjustments", "fixed_time", "TEXT DEFAULT ''")
	return nil
}

func (s *Storage) addColumnIfNotExist(column, colType string) error {
	query := fmt.Sprintf("ALTER TABLE users ADD COLUMN %s %s;", column, colType)
	_, err := s.db.Exec(query)
	return err // Игнорируем ошибку, если колонка уже существует
}

func (s *Storage) addTableColumnIfNotExist(table, column, colType string) error {
	query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", table, column, colType)
	_, err := s.db.Exec(query)
	return err // Игнорируем ошибку, если колонка уже существует
}

// --- Управление пользователями и чатами ---

func (s *Storage) GetUser(chatID int64) (*User, error) {
	query := `SELECT chat_id, city, subscribed_at, daily_schedule_time, COALESCE(evening_schedule_time, ''), COALESCE(broadcast_times, ''), notify_15min, notify_at_time, COALESCE(title, ''), COALESCE(chat_type, 'private'), COALESCE(added_by, 0), COALESCE(custom_footer, ''), COALESCE(custom_header, ''), COALESCE(prayer_names_preset, 'ru'), COALESCE(custom_prayer_names, '') FROM users WHERE chat_id = ?;`
	row := s.db.QueryRow(query, chatID)

	var u User
	err := row.Scan(&u.ChatID, &u.City, &u.SubscribedAt, &u.DailyScheduleTime, &u.EveningScheduleTime, &u.BroadcastTimes, &u.Notify15Min, &u.NotifyAtTime, &u.Title, &u.ChatType, &u.AddedBy, &u.CustomFooter, &u.CustomHeader, &u.PrayerNamesPreset, &u.CustomPrayerNames)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.DailyScheduleTime = normalizeScheduleTime(u.DailyScheduleTime)
	u.EveningScheduleTime = normalizeScheduleTime(u.EveningScheduleTime)
	if u.PrayerNamesPreset == "" {
		u.PrayerNamesPreset = "ru"
	}
	return &u, nil
}

func (s *Storage) SetUserCity(chatID int64, city string) error {
	query := `
	INSERT INTO users (chat_id, city, subscribed_at, daily_schedule_time, evening_schedule_time, broadcast_times, notify_15min, notify_at_time)
	VALUES (?, ?, ?, '06:00', '', '["06:00"]', 1, 1)
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

func (s *Storage) AddBroadcastTime(chatID int64, timeStr string) (string, error) {
	norm, err := NormalizeBroadcastTime(timeStr)
	if err != nil {
		return "", err
	}

	u, _ := s.GetUser(chatID)
	var times []string
	if u != nil {
		times = u.GetParsedBroadcastTimes()
	}

	found := false
	for _, t := range times {
		if t == norm {
			found = true
			break
		}
	}
	if !found {
		times = append(times, norm)
		sort.Strings(times)
	}

	jsonBytes, _ := json.Marshal(times)
	jsonStr := string(jsonBytes)

	query := `
	INSERT INTO users (chat_id, city, subscribed_at, daily_schedule_time, evening_schedule_time, broadcast_times)
	VALUES (?, 'Уфа', ?, '06:00', '', ?)
	ON CONFLICT(chat_id) DO UPDATE SET broadcast_times = excluded.broadcast_times;
	`
	_, err = s.db.Exec(query, chatID, time.Now(), jsonStr)
	return norm, err
}

func (s *Storage) RemoveBroadcastTime(chatID int64, timeStr string) error {
	norm, err := NormalizeBroadcastTime(timeStr)
	if err != nil {
		norm = strings.TrimSpace(timeStr)
	}

	u, _ := s.GetUser(chatID)
	var newTimes []string
	if u != nil {
		for _, t := range u.GetParsedBroadcastTimes() {
			if t != norm {
				newTimes = append(newTimes, t)
			}
		}
	}

	var jsonStr string
	if len(newTimes) > 0 {
		jsonBytes, _ := json.Marshal(newTimes)
		jsonStr = string(jsonBytes)
	} else {
		jsonStr = "[]"
	}

	query := `
	INSERT INTO users (chat_id, city, subscribed_at, daily_schedule_time, evening_schedule_time, broadcast_times)
	VALUES (?, 'Уфа', ?, '', '', ?)
	ON CONFLICT(chat_id) DO UPDATE SET broadcast_times = excluded.broadcast_times, daily_schedule_time = '', evening_schedule_time = '';
	`
	_, err = s.db.Exec(query, chatID, time.Now(), jsonStr)
	return err
}

func (s *Storage) ClearBroadcastTimes(chatID int64) error {
	query := `
	UPDATE users SET broadcast_times = '[]', daily_schedule_time = '', evening_schedule_time = '' WHERE chat_id = ?;
	`
	_, err := s.db.Exec(query, chatID)
	return err
}

func (s *Storage) UpdateDailyTime(chatID int64, dailyTime string) error {
	dailyTime = normalizeScheduleTime(dailyTime)
	query := `
	INSERT INTO users (chat_id, city, subscribed_at, daily_schedule_time, evening_schedule_time, broadcast_times)
	VALUES (?, 'Уфа', ?, ?, '', '["06:00"]')
	ON CONFLICT(chat_id) DO UPDATE SET daily_schedule_time = excluded.daily_schedule_time;
	`
	if _, err := s.db.Exec(query, chatID, time.Now(), dailyTime); err != nil {
		return err
	}
	if dailyTime != "" && dailyTime != "off" {
		_, _ = s.AddBroadcastTime(chatID, dailyTime)
	}
	return nil
}

func (s *Storage) UpdateEveningTime(chatID int64, eveningTime string) error {
	eveningTime = normalizeScheduleTime(eveningTime)
	query := `
	INSERT INTO users (chat_id, city, subscribed_at, daily_schedule_time, evening_schedule_time, broadcast_times)
	VALUES (?, 'Уфа', ?, '06:00', ?, '["06:00"]')
	ON CONFLICT(chat_id) DO UPDATE SET evening_schedule_time = excluded.evening_schedule_time;
	`
	if _, err := s.db.Exec(query, chatID, time.Now(), eveningTime); err != nil {
		return err
	}
	if eveningTime != "" && eveningTime != "off" {
		_, _ = s.AddBroadcastTime(chatID, eveningTime)
	}
	return nil
}

func (s *Storage) ToggleNotify15min(chatID int64) error {
	query := `
	INSERT INTO users (chat_id, city, subscribed_at, daily_schedule_time, evening_schedule_time, broadcast_times, notify_15min, notify_at_time)
	VALUES (?, 'Уфа', ?, '06:00', '', '["06:00"]', 0, 1)
	ON CONFLICT(chat_id) DO UPDATE SET notify_15min = CASE WHEN notify_15min = 1 THEN 0 ELSE 1 END;
	`
	_, err := s.db.Exec(query, chatID, time.Now())
	return err
}

func (s *Storage) ToggleNotifyAtTime(chatID int64) error {
	query := `
	INSERT INTO users (chat_id, city, subscribed_at, daily_schedule_time, evening_schedule_time, broadcast_times, notify_15min, notify_at_time)
	VALUES (?, 'Уфа', ?, '06:00', '', '["06:00"]', 1, 0)
	ON CONFLICT(chat_id) DO UPDATE SET notify_at_time = CASE WHEN notify_at_time = 1 THEN 0 ELSE 1 END;
	`
	_, err := s.db.Exec(query, chatID, time.Now())
	return err
}

func (s *Storage) Subscribe(chatID int64, city string) error {
	query := `
	INSERT INTO users (chat_id, city, subscribed_at, daily_schedule_time, evening_schedule_time, broadcast_times)
	VALUES (?, ?, ?, '06:00', '', '["06:00"]')
	ON CONFLICT(chat_id) DO UPDATE SET 
		city = excluded.city,
		broadcast_times = CASE WHEN broadcast_times = '[]' OR broadcast_times = '' THEN '["06:00"]' ELSE broadcast_times END;
	`
	_, err := s.db.Exec(query, chatID, city, time.Now())
	if err != nil {
		return fmt.Errorf("ошибка добавления пользователя: %w", err)
	}
	return nil
}

func (s *Storage) Unsubscribe(chatID int64) error {
	query := `UPDATE users SET broadcast_times = '[]', daily_schedule_time = '', evening_schedule_time = '' WHERE chat_id = ?;`
	_, err := s.db.Exec(query, chatID)
	if err != nil {
		return fmt.Errorf("ошибка отключения рассылки: %w", err)
	}
	return nil
}

func (s *Storage) SaveChatInfo(chatID int64, title, chatType string, addedBy int64) error {
	query := `
	INSERT INTO users (chat_id, city, subscribed_at, daily_schedule_time, evening_schedule_time, broadcast_times, title, chat_type, added_by)
	VALUES (?, 'Уфа', ?, '06:00', '', '["06:00"]', ?, ?, ?)
	ON CONFLICT(chat_id) DO UPDATE SET 
		title = CASE WHEN excluded.title != '' THEN excluded.title ELSE users.title END,
		chat_type = CASE WHEN excluded.chat_type != '' THEN excluded.chat_type ELSE users.chat_type END,
		added_by = CASE WHEN excluded.added_by != 0 THEN excluded.added_by ELSE users.added_by END;
	`
	_, err := s.db.Exec(query, chatID, time.Now(), title, chatType, addedBy)
	return err
}

func (s *Storage) GetManagedChats(adminID int64, isSuperAdmin bool) ([]User, error) {
	var query string
	var args []interface{}

	if isSuperAdmin {
		query = `SELECT chat_id, city, subscribed_at, daily_schedule_time, COALESCE(evening_schedule_time, ''), COALESCE(broadcast_times, ''), notify_15min, notify_at_time, COALESCE(title, ''), COALESCE(chat_type, 'private'), COALESCE(added_by, 0), COALESCE(custom_footer, ''), COALESCE(custom_header, ''), COALESCE(prayer_names_preset, 'ru'), COALESCE(custom_prayer_names, '') FROM users WHERE chat_id < 0 ORDER BY chat_id DESC;`
	} else {
		query = `SELECT chat_id, city, subscribed_at, daily_schedule_time, COALESCE(evening_schedule_time, ''), COALESCE(broadcast_times, ''), notify_15min, notify_at_time, COALESCE(title, ''), COALESCE(chat_type, 'private'), COALESCE(added_by, 0), COALESCE(custom_footer, ''), COALESCE(custom_header, ''), COALESCE(prayer_names_preset, 'ru'), COALESCE(custom_prayer_names, '') FROM users WHERE chat_id < 0 AND (added_by = ? OR added_by = 0) ORDER BY chat_id DESC;`
		args = append(args, adminID)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ChatID, &u.City, &u.SubscribedAt, &u.DailyScheduleTime, &u.EveningScheduleTime, &u.BroadcastTimes, &u.Notify15Min, &u.NotifyAtTime, &u.Title, &u.ChatType, &u.AddedBy, &u.CustomFooter, &u.CustomHeader, &u.PrayerNamesPreset, &u.CustomPrayerNames); err != nil {
			return nil, err
		}
		u.DailyScheduleTime = normalizeScheduleTime(u.DailyScheduleTime)
		u.EveningScheduleTime = normalizeScheduleTime(u.EveningScheduleTime)
		if u.PrayerNamesPreset == "" {
			u.PrayerNamesPreset = "ru"
		}
		chats = append(chats, u)
	}
	return chats, rows.Err()
}

func (s *Storage) UpdateChatPreset(chatID int64, preset string) error {
	query := `
	INSERT INTO users (chat_id, city, subscribed_at, daily_schedule_time, evening_schedule_time, prayer_names_preset)
	VALUES (?, 'Уфа', ?, '06:00', '', ?)
	ON CONFLICT(chat_id) DO UPDATE SET prayer_names_preset = excluded.prayer_names_preset;
	`
	_, err := s.db.Exec(query, chatID, time.Now(), preset)
	return err
}

func (s *Storage) UpdateChatFooter(chatID int64, footer string) error {
	query := `
	INSERT INTO users (chat_id, city, subscribed_at, daily_schedule_time, evening_schedule_time, custom_footer)
	VALUES (?, 'Уфа', ?, '06:00', '', ?)
	ON CONFLICT(chat_id) DO UPDATE SET custom_footer = excluded.custom_footer;
	`
	_, err := s.db.Exec(query, chatID, time.Now(), footer)
	return err
}

func (s *Storage) UpdateChatHeader(chatID int64, header string) error {
	query := `
	INSERT INTO users (chat_id, city, subscribed_at, daily_schedule_time, evening_schedule_time, custom_header)
	VALUES (?, 'Уфа', ?, '06:00', '', ?)
	ON CONFLICT(chat_id) DO UPDATE SET custom_header = excluded.custom_header;
	`
	_, err := s.db.Exec(query, chatID, time.Now(), header)
	return err
}

func (s *Storage) UpdateCustomPrayerName(chatID int64, prayerKey, customName string) error {
	u, _ := s.GetUser(chatID)
	var names map[string]string
	if u != nil {
		names = ParseCustomPrayerNames(u.CustomPrayerNames)
	} else {
		names = make(map[string]string)
	}

	customName = strings.TrimSpace(customName)
	if customName == "" || customName == "-" {
		delete(names, prayerKey)
	} else {
		names[prayerKey] = customName
	}

	var jsonStr string
	if len(names) > 0 {
		jsonBytes, _ := json.Marshal(names)
		jsonStr = string(jsonBytes)
	}

	query := `
	INSERT INTO users (chat_id, city, subscribed_at, daily_schedule_time, evening_schedule_time, custom_prayer_names)
	VALUES (?, 'Уфа', ?, '06:00', '', ?)
	ON CONFLICT(chat_id) DO UPDATE SET custom_prayer_names = excluded.custom_prayer_names;
	`
	_, err := s.db.Exec(query, chatID, time.Now(), jsonStr)
	return err
}

func (s *Storage) ResetCustomPrayerNames(chatID int64) error {
	query := `UPDATE users SET custom_prayer_names = '' WHERE chat_id = ?;`
	_, err := s.db.Exec(query, chatID)
	return err
}

func (s *Storage) DeleteChat(chatID int64) error {
	query := `DELETE FROM users WHERE chat_id = ?;`
	_, err := s.db.Exec(query, chatID)
	return err
}

func (s *Storage) ResetChatCustomization(chatID int64) error {
	query := `
	UPDATE users SET custom_footer = '', custom_header = '', prayer_names_preset = 'ru', custom_prayer_names = '' WHERE chat_id = ?;
	`
	_, err := s.db.Exec(query, chatID)
	return err
}

func (s *Storage) GetSubscribers() ([]User, error) {
	query := `SELECT chat_id, city, subscribed_at, daily_schedule_time, COALESCE(evening_schedule_time, ''), COALESCE(broadcast_times, ''), notify_15min, notify_at_time, COALESCE(title, ''), COALESCE(chat_type, 'private'), COALESCE(added_by, 0), COALESCE(custom_footer, ''), COALESCE(custom_header, ''), COALESCE(prayer_names_preset, 'ru'), COALESCE(custom_prayer_names, '') FROM users;`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения подписчиков: %w", err)
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ChatID, &u.City, &u.SubscribedAt, &u.DailyScheduleTime, &u.EveningScheduleTime, &u.BroadcastTimes, &u.Notify15Min, &u.NotifyAtTime, &u.Title, &u.ChatType, &u.AddedBy, &u.CustomFooter, &u.CustomHeader, &u.PrayerNamesPreset, &u.CustomPrayerNames); err != nil {
			return nil, err
		}
		u.DailyScheduleTime = normalizeScheduleTime(u.DailyScheduleTime)
		u.EveningScheduleTime = normalizeScheduleTime(u.EveningScheduleTime)
		if u.PrayerNamesPreset == "" {
			u.PrayerNamesPreset = "ru"
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

func (s *Storage) SaveAdjustment(city, prayer string, offsetMinutes int, fixedTime string, validUntil time.Time) error {
	// Сначала удаляем предыдущие правила для того же города и молитвы
	_, _ = s.db.Exec(`DELETE FROM prayer_adjustments WHERE city = ? AND prayer = ?;`, city, prayer)

	query := `
	INSERT INTO prayer_adjustments (city, prayer, offset_minutes, fixed_time, valid_until, created_at)
	VALUES (?, ?, ?, ?, ?, ?);
	`
	_, err := s.db.Exec(query, city, prayer, offsetMinutes, fixedTime, validUntil, time.Now())
	return err
}

func (s *Storage) GetActiveAdjustments(city string, date time.Time) (map[string]api.PrayerRule, error) {
	query := `
	SELECT prayer, offset_minutes, COALESCE(fixed_time, '') FROM prayer_adjustments 
	WHERE city = ? AND valid_until >= ?;
	`
	// Сравниваем с началом текущего дня даты
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	rows, err := s.db.Query(query, city, dayStart)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	adjustments := make(map[string]api.PrayerRule)
	for rows.Next() {
		var prayer string
		var offset int
		var fixed string
		if err := rows.Scan(&prayer, &offset, &fixed); err != nil {
			return nil, err
		}
		adjustments[prayer] = api.PrayerRule{
			OffsetMinutes: offset,
			FixedTime:     fixed,
		}
	}
	return adjustments, rows.Err()
}

func (s *Storage) GetAllActiveAdjustments() ([]PrayerAdjustment, error) {
	query := `
	SELECT id, city, prayer, offset_minutes, COALESCE(fixed_time, ''), valid_until, created_at 
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
		if err := rows.Scan(&a.ID, &a.City, &a.Prayer, &a.OffsetMinutes, &a.FixedTime, &a.ValidUntil, &a.CreatedAt); err != nil {
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


