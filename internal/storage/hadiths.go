package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Hadith struct {
	ID         int64     `json:"id"`
	Text       string    `json:"text"`
	Source     string    `json:"source"`
	Collection string    `json:"collection"` // "bukhari", "muslim", "riyad"
	Category   string    `json:"category"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

const (
	CategoryFastingWhiteDays = "fasting_white_days" // Пост: белые дни (13, 14, 15)
	CategoryFastingMonThu    = "fasting_mon_thu"    // Пост: понедельник и четверг
	CategoryFastingGeneral   = "fasting_general"    // Пост: общие достоинства
	CategoryMonthRamadan     = "month_ramadan"      // Месяц Рамадан
	CategoryMonthShawwal     = "month_shawwal"      // Месяц Шавваль
	CategoryMonthDhulHijjah  = "month_dhul_hijjah"   // Месяц Зуль-Хиджа
	CategoryMonthMuharram    = "month_muharram"     // Месяц Мухаррам
	CategoryMonthRajab       = "month_rajab"        // Месяц Раджаб
	CategoryMonthShaban      = "month_shaban"       // Месяц Шаабан
	CategoryMonthDhulQadah   = "month_dhul_qadah"   // Месяц Зуль-Каада
	CategoryPrayerFajr       = "prayer_fajr"        // Молитва Фаджр
	CategoryPrayerDhuhr      = "prayer_dhuhr"       // Молитва Зухр
	CategoryPrayerAsr        = "prayer_asr"         // Молитва Аср
	CategoryPrayerMaghrib    = "prayer_maghrib"     // Молитва Магриб
	CategoryPrayerIsha       = "prayer_isha"        // Молитва Иша
	CategoryPrayerJumah      = "prayer_jumah"       // Пятничная молитва Джума
	CategoryGeneral          = "general"            // Общие темы (нрав, дуа, благочестие)
)

const (
	CollectionBukhari = "bukhari" // Сахих аль-Бухари
	CollectionMuslim  = "muslim"  // Сахих Муслим
	CollectionRiyad   = "riyad"   // Сады праведных
)

var AllCategories = []string{
	CategoryGeneral,
	CategoryFastingWhiteDays,
	CategoryFastingMonThu,
	CategoryFastingGeneral,
	CategoryMonthRamadan,
	CategoryMonthShawwal,
	CategoryMonthDhulHijjah,
	CategoryMonthMuharram,
	CategoryMonthRajab,
	CategoryMonthShaban,
	CategoryMonthDhulQadah,
	CategoryPrayerFajr,
	CategoryPrayerDhuhr,
	CategoryPrayerAsr,
	CategoryPrayerMaghrib,
	CategoryPrayerIsha,
	CategoryPrayerJumah,
}

var AllCollections = []string{
	CollectionBukhari,
	CollectionMuslim,
	CollectionRiyad,
}

func GetCategoryTitle(category string) string {
	switch category {
	case CategoryFastingWhiteDays:
		return "🌕 Пост: Белые дни (13-15)"
	case CategoryFastingMonThu:
		return "📅 Пост: Пн / Чт"
	case CategoryFastingGeneral:
		return "✨ Пост: Общие достоинства"
	case CategoryMonthRamadan:
		return "🌙 Месяц: Рамадан"
	case CategoryMonthShawwal:
		return "🌙 Месяц: Шавваль"
	case CategoryMonthDhulHijjah:
		return "🕋 Месяц: Зуль-Хиджа"
	case CategoryMonthMuharram:
		return "🌙 Месяц: Мухаррам"
	case CategoryMonthRajab:
		return "🌙 Месяц: Раджаб"
	case CategoryMonthShaban:
		return "🌙 Месяц: Шаабан"
	case CategoryMonthDhulQadah:
		return "🌙 Месяц: Зуль-Каада"
	case CategoryPrayerFajr:
		return "🌅 Молитва: Фаджр"
	case CategoryPrayerDhuhr:
		return "☀️ Молитва: Зухр"
	case CategoryPrayerAsr:
		return "🌤 Молитва: Аср"
	case CategoryPrayerMaghrib:
		return "🌆 Молитва: Магриб"
	case CategoryPrayerIsha:
		return "🌙 Молитва: Иша"
	case CategoryPrayerJumah:
		return "🕌 Молитва: Джума (Пятница)"
	case CategoryGeneral:
		return "📖 Общие темы (нрав, дуа, искренность)"
	default:
		return category
	}
}

func GetCollectionTitle(collection string) string {
	switch collection {
	case CollectionBukhari:
		return "Сахих аль-Бухари"
	case CollectionMuslim:
		return "Сахих Муслим"
	case CollectionRiyad:
		return "Сады праведных"
	default:
		return collection
	}
}

// AddHadith добавляет новый хадис в базу данных
func (s *Storage) AddHadith(text, source, collection, category string) (int64, error) {
	text = strings.TrimSpace(text)
	source = strings.TrimSpace(source)
	collection = strings.TrimSpace(collection)
	category = strings.TrimSpace(category)

	if text == "" {
		return 0, fmt.Errorf("текст хадиса не может быть пустым")
	}
	if source == "" {
		return 0, fmt.Errorf("источник хадиса не может быть пустым")
	}
	if collection == "" {
		collection = CollectionRiyad
	}
	if category == "" {
		category = CategoryGeneral
	}

	query := `INSERT INTO hadiths (text, source, collection, category, is_active, created_at) VALUES (?, ?, ?, ?, 1, ?);`
	res, err := s.db.Exec(query, text, source, collection, category, time.Now())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// DeleteHadith удаляет хадис по ID
func (s *Storage) DeleteHadith(id int64) error {
	query := `DELETE FROM hadiths WHERE id = ?;`
	_, err := s.db.Exec(query, id)
	return err
}

// ToggleHadithActive переключает активность хадиса
func (s *Storage) ToggleHadithActive(id int64) error {
	query := `UPDATE hadiths SET is_active = CASE WHEN is_active = 1 THEN 0 ELSE 1 END WHERE id = ?;`
	_, err := s.db.Exec(query, id)
	return err
}

// GetHadithByID возвращает хадис по ID
func (s *Storage) GetHadithByID(id int64) (*Hadith, error) {
	query := `SELECT id, text, source, collection, category, is_active, created_at FROM hadiths WHERE id = ?;`
	row := s.db.QueryRow(query, id)

	var h Hadith
	err := row.Scan(&h.ID, &h.Text, &h.Source, &h.Collection, &h.Category, &h.IsActive, &h.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// GetHadiths возвращает список хадисов с пагинацией и опциональным фильтром по категории
func (s *Storage) GetHadiths(category string, limit, offset int) ([]Hadith, int, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	var countQuery string
	var selectQuery string
	var args []interface{}
	var countArgs []interface{}

	if category != "" && category != "all" {
		countQuery = `SELECT COUNT(*) FROM hadiths WHERE category = ?;`
		countArgs = append(countArgs, category)
		selectQuery = `SELECT id, text, source, collection, category, is_active, created_at FROM hadiths WHERE category = ? ORDER BY id DESC LIMIT ? OFFSET ?;`
		args = append(args, category, limit, offset)
	} else {
		countQuery = `SELECT COUNT(*) FROM hadiths;`
		selectQuery = `SELECT id, text, source, collection, category, is_active, created_at FROM hadiths ORDER BY id DESC LIMIT ? OFFSET ?;`
		args = append(args, limit, offset)
	}

	var total int
	if err := s.db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Query(selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []Hadith
	for rows.Next() {
		var h Hadith
		if err := rows.Scan(&h.ID, &h.Text, &h.Source, &h.Collection, &h.Category, &h.IsActive, &h.CreatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, h)
	}

	return list, total, rows.Err()
}

// GetRandomHadith возвращает случайный активный хадис по категории
func (s *Storage) GetRandomHadith(category string) (*Hadith, error) {
	query := `SELECT id, text, source, collection, category, is_active, created_at FROM hadiths WHERE category = ? AND is_active = 1 ORDER BY RANDOM() LIMIT 1;`
	row := s.db.QueryRow(query, category)

	var h Hadith
	err := row.Scan(&h.ID, &h.Text, &h.Source, &h.Collection, &h.Category, &h.IsActive, &h.CreatedAt)
	if err == sql.ErrNoRows {
		// Если по конкретной категории ничего нет, пробуем CategoryGeneral или любой активный
		return s.GetRandomHadithAny()
	}
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// GetRandomHadithAny возвращает любой активный случайный хадис
func (s *Storage) GetRandomHadithAny() (*Hadith, error) {
	query := `SELECT id, text, source, collection, category, is_active, created_at FROM hadiths WHERE is_active = 1 ORDER BY RANDOM() LIMIT 1;`
	row := s.db.QueryRow(query)

	var h Hadith
	err := row.Scan(&h.ID, &h.Text, &h.Source, &h.Collection, &h.Category, &h.IsActive, &h.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// GetHadithsStats возвращает общую статистику хадисов
func (s *Storage) GetHadithsStats() (total int, active int, byCat map[string]int, byCol map[string]int, err error) {
	byCat = make(map[string]int)
	byCol = make(map[string]int)

	_ = s.db.QueryRow(`SELECT COUNT(*) FROM hadiths;`).Scan(&total)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM hadiths WHERE is_active = 1;`).Scan(&active)

	rowsCat, err := s.db.Query(`SELECT category, COUNT(*) FROM hadiths WHERE is_active = 1 GROUP BY category;`)
	if err == nil {
		for rowsCat.Next() {
			var cat string
			var count int
			if errScan := rowsCat.Scan(&cat, &count); errScan == nil {
				byCat[cat] = count
			}
		}
		_ = rowsCat.Close()
		_ = rowsCat.Err()
	}

	rowsCol, err := s.db.Query(`SELECT collection, COUNT(*) FROM hadiths WHERE is_active = 1 GROUP BY collection;`)
	if err == nil {
		for rowsCol.Next() {
			var col string
			var count int
			if errScan := rowsCol.Scan(&col, &count); errScan == nil {
				byCol[col] = count
			}
		}
		_ = rowsCol.Close()
		_ = rowsCol.Err()
	}

	return total, active, byCat, byCol, nil
}
