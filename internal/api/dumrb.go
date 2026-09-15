package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
}

func NewClient() *Client {
	return &Client{
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		voshodClient: NewVoshodClient(),
	}
}

// FetchPrayerTimes делает запрос к официальному API ДУМ РБ для времен намаза и к voshod-solnca.ru для времени восхода
func (c *Client) FetchPrayerTimes(city string, date time.Time) (*DUMRBItem, error) {
	year := date.Year()
	month := int(date.Month())
	day := date.Day()

	var reqURL string
	if id, ok := CityIDMap[city]; ok && id > 0 {
		reqURL = fmt.Sprintf("https://api.dumrb.com/Prayer/GetByCity?cityId=%d&year=%d&month=%d", id, year, month)
	} else {
		reqURL = fmt.Sprintf("https://api.dumrb.com/Prayer/GetByCity?cityName=%s&year=%d&month=%d", url.QueryEscape(city), year, month)
	}

	resp, err := c.httpClient.Get(reqURL)

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

	// Массив prayerTimes индексируется с 0, где 0 — это 1-е число месяца
	if day < 1 || day > len(apiResp.PrayerTimes) {
		return nil, fmt.Errorf("день %d вне диапазона дней месяца", day)
	}

	todayTiming := apiResp.PrayerTimes[day-1]

	// Получаем время восхода солнца с сайта voshod-solnca.ru
	if c.voshodClient != nil {
		sunrise, err := c.voshodClient.GetSunriseTime(city, date)
		if err == nil && sunrise != "" {
			todayTiming.Sunrise = sunrise
		}
	}

	return &todayTiming, nil
}

// FormatMessage форматирует данные о расписании намаза (ДУМ РБ) и восходе (voshod-solnca.ru)
func FormatMessage(item *DUMRBItem, city string, date time.Time) string {
	dateStr := date.Format("02.01.2006")

	return fmt.Sprintf(
		"🕌 *Расписание намаза*\n"+
			"📅 *Дата:* %s\n"+
			"📍 *Город/Район:* %s\n\n"+
			"🌅 *Фаджр (Конец Сухура):* %s\n"+
			"☀️ *Восход (voshod-solnca.ru):* %s\n"+
			"☀️ *Зухр:* %s\n"+
			"🌤 *Аср:* %s\n"+
			"🌆 *Магриб:* %s\n"+
			"🌙 *Иша:* %s\n\n"+
			"ℹ️ _Времена намазов: ДУМ РБ_\n"+
			"ℹ️ _Восход солнца: voshod-solnca.ru_",
		dateStr,
		city,
		item.Fajr,
		item.Sunrise,
		item.Dhuhr,
		item.Asr,
		item.Maghrib,
		item.Isha,
	)
}
