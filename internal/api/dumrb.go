package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// DUMRBItem описывает время намаза на один день из ответа ДУМ РБ
type DUMRBItem struct {
	SuhurDo string `json:"suhurDo"`
	Fajr    string `json:"fajr"`
	Sunrise string `json:"sunrise"`
	Dhuhr   string `json:"dhuhr"`
	Zaual   string `json:"zaual"`
	Asr     string `json:"asr"`
	Maghrib string `json:"maghrib"`
	Isha    string `json:"isha"`
}

type CityInfo struct {
	ID          int
	CleanName   string
	DisplayName string
	IsCity      bool
	Population  int
}

// BashkortostanCityList содержит города и районы Башкортостана из API ДУМ РБ, отсортированные по убыванию населения
var BashkortostanCityList = []CityInfo{
	// 21 Город
	{ID: 1, CleanName: "Уфа", DisplayName: "Уфа", IsCity: true, Population: 1157994},
	{ID: 23, CleanName: "Стерлитамак", DisplayName: "Стерлитамак", IsCity: true, Population: 277410},
	{ID: 21, CleanName: "Салават", DisplayName: "Салават", IsCity: true, Population: 148575},
	{ID: 18, CleanName: "Нефтекамск", DisplayName: "Нефтекамск", IsCity: true, Population: 131942},
	{ID: 19, CleanName: "Октябрьский", DisplayName: "Октябрьский", IsCity: true, Population: 115557},
	{ID: 24, CleanName: "Туймазы", DisplayName: "Туймазы", IsCity: true, Population: 68349},
	{ID: 8, CleanName: "Белорецк", DisplayName: "Белорецк", IsCity: true, Population: 64525},
	{ID: 12, CleanName: "Ишимбай", DisplayName: "Ишимбай", IsCity: true, Population: 64041},
	{ID: 7, CleanName: "Белебей", DisplayName: "Белебей", IsCity: true, Population: 59195},
	{ID: 22, CleanName: "Сибай", DisplayName: "Сибай", IsCity: true, Population: 56514},
	{ID: 14, CleanName: "Кумертау", DisplayName: "Кумертау", IsCity: true, Population: 56380},
	{ID: 16, CleanName: "Мелеуз", DisplayName: "Мелеуз", IsCity: true, Population: 56152},
	{ID: 2, CleanName: "Бирск", DisplayName: "Бирск", IsCity: true, Population: 46330},
	{ID: 25, CleanName: "Учалы", DisplayName: "Учалы", IsCity: true, Population: 37710},
	{ID: 9, CleanName: "Благовещенск", DisplayName: "Благовещенск", IsCity: true, Population: 35481},
	{ID: 11, CleanName: "Дюртюли", DisplayName: "Дюртюли", IsCity: true, Population: 31274},
	{ID: 27, CleanName: "Янаул", DisplayName: "Янаул", IsCity: true, Population: 25747},
	{ID: 10, CleanName: "Давлеканово", DisplayName: "Давлеканово", IsCity: true, Population: 23696},
	{ID: 6, CleanName: "Баймак", DisplayName: "Баймак", IsCity: true, Population: 17833},
	{ID: 15, CleanName: "Межгорье", DisplayName: "Межгорье", IsCity: true, Population: 15691},
	{ID: 5, CleanName: "Агидель", DisplayName: "Агидель", IsCity: true, Population: 14219},

	// 40 Районов (с указанием райцентра)
	{ID: 50, CleanName: "Иглино", DisplayName: "Иглинский р-н (Иглино)", IsCity: false, Population: 31169},
	{ID: 26, CleanName: "Чишмы", DisplayName: "Чишминский р-н (Чишмы)", IsCity: false, Population: 22597},
	{ID: 33, CleanName: "Раевский", DisplayName: "Альшеевский р-н (Раевский)", IsCity: false, Population: 18976},
	{ID: 67, CleanName: "Чекмагуш", DisplayName: "Чекмагушевский р-н (Чекмагуш)", IsCity: false, Population: 13073},
	{ID: 46, CleanName: "Месягутово", DisplayName: "Дуванский р-н (Месягутово)", IsCity: false, Population: 12019},
	{ID: 45, CleanName: "Красноусольский", DisplayName: "Гафурийский р-н (Красноусольский)", IsCity: false, Population: 11241},
	{ID: 13, CleanName: "Кармаскалы", DisplayName: "Кармаскалинский р-н (Кармаскалы)", IsCity: false, Population: 11076},
	{ID: 57, CleanName: "Кушнаренково", DisplayName: "Кушнаренковский р-н (Кушнаренково)", IsCity: false, Population: 10715},
	{ID: 36, CleanName: "Толбазы", DisplayName: "Аургазинский р-н (Толбазы)", IsCity: false, Population: 10608},
	{ID: 42, CleanName: "Буздяк", DisplayName: "Буздякский р-н (Буздяк)", IsCity: false, Population: 10323},
	{ID: 51, CleanName: "Верхнеяркеево", DisplayName: "Илишевский р-н (Верхнеяркеево)", IsCity: false, Population: 9703},
	{ID: 43, CleanName: "Бураево", DisplayName: "Бураевский р-н (Бураево)", IsCity: false, Population: 9522},
	{ID: 37, CleanName: "Бакалы", DisplayName: "Бакалинский р-н (Бакалы)", IsCity: false, Population: 9514},
	{ID: 32, CleanName: "Аскарово", DisplayName: "Абзелиловский р-н (Аскарово)", IsCity: false, Population: 9208},
	{ID: 66, CleanName: "Акъяр", DisplayName: "Хайбуллинский р-н (Акъяр)", IsCity: false, Population: 8703},
	{ID: 56, CleanName: "Мраково", DisplayName: "Кугарчинский р-н (Мраково)", IsCity: false, Population: 8690},
	{ID: 41, CleanName: "Языково", DisplayName: "Благоварский р-н (Языково)", IsCity: false, Population: 8375},
	{ID: 59, CleanName: "Большеустьикинское", DisplayName: "Мечетлинский р-н (Большеустьикинское)", IsCity: false, Population: 7839},
	{ID: 61, CleanName: "Киргиз-Мияки", DisplayName: "Миякинский р-н (Киргиз-Мияки)", IsCity: false, Population: 7473},
	{ID: 48, CleanName: "Исянгулово", DisplayName: "Зианчуринский р-н (Исянгулово)", IsCity: false, Population: 7418},
	{ID: 35, CleanName: "Аскино", DisplayName: "Аскинский р-н (Аскино)", IsCity: false, Population: 6918},
	{ID: 64, CleanName: "Верхние Татышлы", DisplayName: "Татышлинский р-н (Татышлы)", IsCity: false, Population: 6645},
	{ID: 54, CleanName: "Верхние Киги", DisplayName: "Кигинский р-н (Киги)", IsCity: false, Population: 6629},
	{ID: 55, CleanName: "Николо-Березовка", DisplayName: "Краснокамский р-н (Николо-Березовка)", IsCity: false, Population: 6202},
	{ID: 60, CleanName: "Мишкино", DisplayName: "Мишкинский р-н (Мишкино)", IsCity: false, Population: 6021},
	{ID: 53, CleanName: "Караидель", DisplayName: "Караидельский р-н (Караидель)", IsCity: false, Population: 5980},
	{ID: 34, CleanName: "Архангельское", DisplayName: "Архангельский р-н (Архангельское)", IsCity: false, Population: 5961},
	{ID: 39, CleanName: "Новобелокатай", DisplayName: "Белокатайский р-н (Новобелокатай)", IsCity: false, Population: 5961},
	{ID: 40, CleanName: "Бижбуляк", DisplayName: "Бижбулякский р-н (Бижбуляк)", IsCity: false, Population: 5957},
	{ID: 63, CleanName: "Стерлибашево", DisplayName: "Стерлибашевский р-н (Стерлибашево)", IsCity: false, Population: 5930},
	{ID: 68, CleanName: "Шаран", DisplayName: "Шаранский р-н (Шаран)", IsCity: false, Population: 5929},
	{ID: 58, CleanName: "Ермолаево", DisplayName: "Куюргазинский р-н (Ермолаево)", IsCity: false, Population: 5623},
	{ID: 38, CleanName: "Старобалтачево", DisplayName: "Балтачевский р-н (Старобалтачево)", IsCity: false, Population: 5598},
	{ID: 49, CleanName: "Зилаир", DisplayName: "Зилаирский р-н (Зилаир)", IsCity: false, Population: 5585},
	{ID: 44, CleanName: "Старосубхангулово", DisplayName: "Бурзянский р-н (Старосубхангулово)", IsCity: false, Population: 5245},
	{ID: 52, CleanName: "Калтасы", DisplayName: "Калтасинский р-н (Калтасы)", IsCity: false, Population: 4418},
	{ID: 65, CleanName: "Федоровка", DisplayName: "Фёдоровский р-н (Фёдоровка)", IsCity: false, Population: 4306},
	{ID: 62, CleanName: "Красная Горка", DisplayName: "Нуримановский р-н (Красная Горка)", IsCity: false, Population: 4291},
	{ID: 69, CleanName: "Малояз", DisplayName: "Салаватский р-н (Малояз)", IsCity: false, Population: 4169},
	{ID: 47, CleanName: "Ермекеево", DisplayName: "Ермекеевский р-н (Ермекеево)", IsCity: false, Population: 3910},
}

func GetCitiesOnly() []CityInfo {
	var list []CityInfo
	for _, c := range BashkortostanCityList {
		if c.IsCity {
			list = append(list, c)
		}
	}
	return list
}

func GetDistrictsOnly() []CityInfo {
	var list []CityInfo
	for _, c := range BashkortostanCityList {
		if !c.IsCity {
			list = append(list, c)
		}
	}
	return list
}

var BashkortostanCities []string
var CityIDMap map[string]int
var CityByIDMap map[int]CityInfo

func init() {
	CityIDMap = make(map[string]int)
	CityByIDMap = make(map[int]CityInfo)
	for _, c := range BashkortostanCityList {
		BashkortostanCities = append(BashkortostanCities, c.DisplayName)
		CityIDMap[c.DisplayName] = c.ID
		CityIDMap[c.CleanName] = c.ID
		CityByIDMap[c.ID] = c
	}
}

func GetCityByID(id int) (CityInfo, bool) {
	c, ok := CityByIDMap[id]
	return c, ok
}

type DUMRBResponse struct {
	CityID      int         `json:"cityId"`
	CityName    string      `json:"cityName"`
	Year        int         `json:"year"`
	Month       int         `json:"month"`
	PrayerTimes []DUMRBItem `json:"prayerTimes"`
	Error       string      `json:"error,omitempty"`
}

type Client struct {
	httpClient   *http.Client
	voshodClient *VoshodClient
	storage      AdjustmentStorage
	monthCache   map[string][]DUMRBItem
	cacheMutex   sync.RWMutex
}

func NewClient() *Client {
	return &Client{
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		voshodClient: NewVoshodClient(),
		monthCache:   make(map[string][]DUMRBItem),
	}
}

// SetStorage устанавливает хранилище для чтения активных корректировок
func (c *Client) SetStorage(store AdjustmentStorage) {
	c.storage = store
}

// FetchPrayerTimes делает запрос к официальному API ДУМ РБ для времен намаза и к voshod-solnca.ru для времени восхода
func (c *Client) FetchPrayerTimes(city string, date time.Time) (*DUMRBItem, error) {
	year := date.Year()
	month := int(date.Month())
	day := date.Day()

	cleanCity := cleanCityName(city)
	cacheKey := fmt.Sprintf("%s_%d_%d", cleanCity, year, month)

	c.cacheMutex.RLock()
	cachedMonth, inCache := c.monthCache[cacheKey]
	c.cacheMutex.RUnlock()

	var monthItems []DUMRBItem

	if inCache {
		monthItems = cachedMonth
	} else {
		var reqURL string
		if id, ok := CityIDMap[city]; ok && id > 0 {
			reqURL = fmt.Sprintf("https://api.dumrb.com/Prayer/GetByCity?cityId=%d&year=%d&month=%d", id, year, month)
		} else {
			reqURL = fmt.Sprintf("https://api.dumrb.com/Prayer/GetByCity?cityName=%s&year=%d&month=%d", url.QueryEscape(city), year, month)
		}

		req, err := http.NewRequest("GET", reqURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "application/json, text/plain, */*")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("ошибка HTTP-запроса к dumrb: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("неуспешный статус ответа dumrb: %d", resp.StatusCode)
		}

		var apiResp DUMRBResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return nil, fmt.Errorf("ошибка декодирования JSON: %w", err)
		}

		if apiResp.Error != "" {
			return nil, fmt.Errorf("ошибка API ДУМ РБ: %s", apiResp.Error)
		}

		monthItems = apiResp.PrayerTimes
		if len(monthItems) > 0 {
			c.cacheMutex.Lock()
			c.monthCache[cacheKey] = monthItems
			c.cacheMutex.Unlock()
		}
	}

	// Массив prayerTimes индексируется с 0, где 0 — это 1-е число месяца
	if day < 1 || day > len(monthItems) {
		return nil, fmt.Errorf("день %d вне диапазона дней месяца", day)
	}

	todayTiming := monthItems[day-1]

	// 1. Получаем время восхода солнца с сайта voshod-solnca.ru
	if c.voshodClient != nil {
		sunrise, err := c.voshodClient.GetSunriseTime(city, date)
		if err == nil && sunrise != "" {
			todayTiming.Sunrise = sunrise
		}
	}

	// 2. Применяем активные корректировки из БД (если настроены)
	if c.storage != nil {
		cleanCity := cleanCityName(city)
		adjustments, err := c.storage.GetActiveAdjustments(cleanCity, date)
		if err == nil && len(adjustments) > 0 {
			ApplyPrayerAdjustments(&todayTiming, adjustments)
		}
	}

	return &todayTiming, nil
}

// PrayerLabels содержит локализованные и кастомизированные заголовки молитв
type PrayerLabels struct {
	Header         string
	DateLabel      string
	CityLabel      string
	Fajr           string
	Sunrise        string
	Dhuhr          string
	Asr            string
	Maghrib        string
	Isha           string
	SourceNote     string
	EveningHeader  string
	RemainingTitle string
	TomorrowTitle  string
}

// PrayerFormatConfig определяет параметры форматирования расписания
type PrayerFormatConfig struct {
	Preset        string            // "ru", "ar", "ru_ar", "ba"
	CustomHeader  string
	CustomFooter  string
	CustomPrayers map[string]string // "fajr", "sunrise", "dhuhr", "asr", "maghrib", "isha"
}

func applyPrayerOverrides(labels PrayerLabels, custom map[string]string) PrayerLabels {
	if len(custom) == 0 {
		return labels
	}
	if v, ok := custom["fajr"]; ok && strings.TrimSpace(v) != "" {
		labels.Fajr = formatPrayerLabel(v)
	}
	if v, ok := custom["sunrise"]; ok && strings.TrimSpace(v) != "" {
		labels.Sunrise = formatPrayerLabel(v)
	}
	if v, ok := custom["dhuhr"]; ok && strings.TrimSpace(v) != "" {
		labels.Dhuhr = formatPrayerLabel(v)
	}
	if v, ok := custom["asr"]; ok && strings.TrimSpace(v) != "" {
		labels.Asr = formatPrayerLabel(v)
	}
	if v, ok := custom["maghrib"]; ok && strings.TrimSpace(v) != "" {
		labels.Maghrib = formatPrayerLabel(v)
	}
	if v, ok := custom["isha"]; ok && strings.TrimSpace(v) != "" {
		labels.Isha = formatPrayerLabel(v)
	}
	return labels
}

func formatPrayerLabel(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasSuffix(s, ":") {
		s += ":"
	}
	if !strings.Contains(s, "*") {
		s = "*" + s + "*"
	}
	return s
}

// GetPrayerLabels возвращает набор подписей для заданного пресета
func GetPrayerLabels(preset string) PrayerLabels {
	switch preset {
	case "ar":
		return PrayerLabels{
			Header:         "🕌 *مواقيت الصلاة*",
			DateLabel:      "📅 *التاريخ:*",
			CityLabel:      "📍 *المدينة/المنطقة:*",
			Fajr:           "🌅 *الفجر:*",
			Sunrise:        "☀️ *الشروق:*",
			Dhuhr:          "☀️ *الظهر:*",
			Asr:            "🌤 *العصر:*",
			Maghrib:        "🌆 *المغرب:*",
			Isha:           "🌙 *العشاء:*",
			SourceNote:     "ℹ️ _مواقيت الصلاة: الإدارة الدينية_",
			EveningHeader:  "🌙 *مواقيت الصلاة للمساء والغد*",
			RemainingTitle: "📌 *الصلوات المتبقية اليوم (%s):*",
			TomorrowTitle:  "📅 *جدول الغد (%s):*",
		}
	case "ru_ar":
		return PrayerLabels{
			Header:         "🕌 *Расписание намаза | مواقيت الصلاة*",
			DateLabel:      "📅 *Дата:*",
			CityLabel:      "📍 *Город/Район:*",
			Fajr:           "🌅 *Фаджр (الفجر):*",
			Sunrise:        "☀️ *Восход (الشروق):*",
			Dhuhr:          "☀️ *Зухр (الظهر):*",
			Asr:            "🌤 *Аср (العصر):*",
			Maghrib:        "🌆 *Магриб (المغرب):*",
			Isha:           "🌙 *Иша (العشاء):*",
			SourceNote:     "ℹ️ _Времена намазов: ДУМ РБ_",
			EveningHeader:  "🌙 *Вечерняя рассылка расписания намаза*",
			RemainingTitle: "📌 *Оставшиеся намазы на сегодня (%s):*",
			TomorrowTitle:  "📅 *Расписание на завтра (%s):*",
		}
	case "ba":
		return PrayerLabels{
			Header:         "🕌 *Намаҙ ваҡыттары*",
			DateLabel:      "📅 *Көн:*",
			CityLabel:      "📍 *Ҡала/Район:*",
			Fajr:           "🌅 *Иртәнге:*",
			Sunrise:        "☀️ *Кояш сығыуы:*",
			Dhuhr:          "☀️ *Өйлә:*",
			Asr:            "🌤 *Икенде:*",
			Maghrib:        "🌆 *Аҡшам:*",
			Isha:           "🌙 *Йәстү:*",
			SourceNote:     "ℹ️ _Намаҙ ваҡыттары: БР МДН_",
			EveningHeader:  "🌙 *Кичке намаҙ ваҡыттары*",
			RemainingTitle: "📌 *Бөгөнгә ҡалған намаҙҙар (%s):*",
			TomorrowTitle:  "📅 *Иртәгәгә расписание (%s):*",
		}
	default: // "ru"
		return PrayerLabels{
			Header:         "🕌 *Расписание намаза*",
			DateLabel:      "📅 *Дата:*",
			CityLabel:      "📍 *Город/Район:*",
			Fajr:           "🌅 *Фаджр:*",
			Sunrise:        "☀️ *Восход:*",
			Dhuhr:          "☀️ *Зухр:*",
			Asr:            "🌤 *Аср:*",
			Maghrib:        "🌆 *Магриб:*",
			Isha:           "🌙 *Иша:*",
			SourceNote:     "ℹ️ _Времена намазов: ДУМ РБ_",
			EveningHeader:  "🌙 *Вечерняя рассылка расписания намаза*",
			RemainingTitle: "📌 *Оставшиеся намазы на сегодня (%s):*",
			TomorrowTitle:  "📅 *Расписание на завтра (%s):*",
		}
	}
}

// FormatMessage форматирует данные о расписании намаза по умолчанию
func FormatMessage(item *DUMRBItem, city string, date time.Time) string {
	return FormatMessageCustom(item, city, date, PrayerFormatConfig{Preset: "ru"})
}

// FormatMessageCustom форматирует данные с учетом пресета и кастомного оформления
func FormatMessageCustom(item *DUMRBItem, city string, date time.Time, cfg PrayerFormatConfig) string {
	labels := GetPrayerLabels(cfg.Preset)
	labels = applyPrayerOverrides(labels, cfg.CustomPrayers)

	header := labels.Header
	if strings.TrimSpace(cfg.CustomHeader) != "" {
		header = strings.TrimSpace(cfg.CustomHeader)
	}

	dateStr := date.Format("02.01.2006")

	text := fmt.Sprintf(
		"%s\n"+
			"%s %s\n"+
			"%s %s\n\n"+
			"%s %s\n"+
			"%s %s\n"+
			"%s %s\n"+
			"%s %s\n"+
			"%s %s\n"+
			"%s %s\n\n"+
			"%s",
		header,
		labels.DateLabel, dateStr,
		labels.CityLabel, city,
		labels.Fajr, item.Fajr,
		labels.Sunrise, item.Sunrise,
		labels.Dhuhr, item.Dhuhr,
		labels.Asr, item.Asr,
		labels.Maghrib, item.Maghrib,
		labels.Isha, item.Isha,
		labels.SourceNote,
	)

	if strings.TrimSpace(cfg.CustomFooter) != "" {
		text += "\n\n" + strings.TrimSpace(cfg.CustomFooter)
	}

	return text
}

// FormatEveningMessage форматирует Вечернюю рассылку по умолчанию
func FormatEveningMessage(todayItem *DUMRBItem, tomorrowItem *DUMRBItem, city string, today time.Time) string {
	return FormatEveningMessageCustom(todayItem, tomorrowItem, city, today, PrayerFormatConfig{Preset: "ru"})
}

// FormatEveningMessageCustom форматирует Вечернюю рассылку с учетом пресета и кастомного оформления
func FormatEveningMessageCustom(todayItem *DUMRBItem, tomorrowItem *DUMRBItem, city string, today time.Time, cfg PrayerFormatConfig) string {
	labels := GetPrayerLabels(cfg.Preset)
	labels = applyPrayerOverrides(labels, cfg.CustomPrayers)

	header := labels.EveningHeader
	if strings.TrimSpace(cfg.CustomHeader) != "" {
		header = strings.TrimSpace(cfg.CustomHeader)
	}

	tomorrow := today.AddDate(0, 0, 1)
	todayStr := today.Format("02.01.2006")
	tomorrowStr := tomorrow.Format("02.01.2006")

	remTitle := fmt.Sprintf(labels.RemainingTitle, todayStr)
	tomTitle := fmt.Sprintf(labels.TomorrowTitle, tomorrowStr)

	text := fmt.Sprintf(
		"%s\n"+
			"%s %s\n\n"+
			"%s\n"+
			"%s %s\n"+
			"%s %s\n\n"+
			"%s\n"+
			"%s %s\n"+
			"%s %s\n"+
			"%s %s\n"+
			"%s %s\n"+
			"%s %s\n"+
			"%s %s\n\n"+
			"%s",
		header,
		labels.CityLabel, city,
		remTitle,
		labels.Maghrib, todayItem.Maghrib,
		labels.Isha, todayItem.Isha,
		tomTitle,
		labels.Fajr, tomorrowItem.Fajr,
		labels.Sunrise, tomorrowItem.Sunrise,
		labels.Dhuhr, tomorrowItem.Dhuhr,
		labels.Asr, tomorrowItem.Asr,
		labels.Maghrib, tomorrowItem.Maghrib,
		labels.Isha, tomorrowItem.Isha,
		labels.SourceNote,
	)

	if strings.TrimSpace(cfg.CustomFooter) != "" {
		text += "\n\n" + strings.TrimSpace(cfg.CustomFooter)
	}

	return text
}

