package api

import (
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// SettlementCoordinates хранит координаты населенного пункта для fallback-расчета восхода
type SettlementCoordinates struct {
	Lat float64
	Lon float64
}

// CitySlugsAndCoords содержит соответствие населенных пунктов Башкортостана slugs на сайте voshod-solnca.ru и координатам
var CitySlugsAndCoords = map[string]struct {
	Slug   string
	Coords SettlementCoordinates
}{
	// 21 Город
	"Уфа":             {Slug: "уфа", Coords: SettlementCoordinates{Lat: 54.7388, Lon: 55.9721}},
	"Стерлитамак":     {Slug: "стерлитамак", Coords: SettlementCoordinates{Lat: 53.6246, Lon: 55.9501}},
	"Салават":         {Slug: "салават", Coords: SettlementCoordinates{Lat: 53.3614, Lon: 55.9242}},
	"Нефтекамск":      {Slug: "нефтекамск", Coords: SettlementCoordinates{Lat: 56.0906, Lon: 54.2678}},
	"Октябрьский":     {Slug: "октябрьский_(башкортостан)", Coords: SettlementCoordinates{Lat: 54.4815, Lon: 53.4710}},
	"Туймазы":         {Slug: "туймазы", Coords: SettlementCoordinates{Lat: 54.6074, Lon: 53.7088}},
	"Белорецк":        {Slug: "белорецк", Coords: SettlementCoordinates{Lat: 53.9688, Lon: 58.4069}},
	"Ишимбай":         {Slug: "ишимбай", Coords: SettlementCoordinates{Lat: 53.4500, Lon: 56.0333}},
	"Белебей":         {Slug: "белебей", Coords: SettlementCoordinates{Lat: 54.1167, Lon: 54.1167}},
	"Сибай":           {Slug: "сибай", Coords: SettlementCoordinates{Lat: 52.7167, Lon: 58.6500}},
	"Кумертау":        {Slug: "кумертау", Coords: SettlementCoordinates{Lat: 52.7667, Lon: 55.7833}},
	"Мелеуз":          {Slug: "мелеуз", Coords: SettlementCoordinates{Lat: 52.9667, Lon: 55.9333}},
	"Бирск":           {Slug: "бирск", Coords: SettlementCoordinates{Lat: 55.4167, Lon: 55.5333}},
	"Учалы":           {Slug: "учалы", Coords: SettlementCoordinates{Lat: 54.3167, Lon: 59.4500}},
	"Благовещенск":    {Slug: "благовещенск_(башкортостан)", Coords: SettlementCoordinates{Lat: 55.0333, Lon: 55.9833}},
	"Дюртюли":         {Slug: "дюртюли", Coords: SettlementCoordinates{Lat: 55.4833, Lon: 54.8667}},
	"Янаул":           {Slug: "янаул", Coords: SettlementCoordinates{Lat: 56.2667, Lon: 54.9333}},
	"Давлеканово":     {Slug: "давлеканово", Coords: SettlementCoordinates{Lat: 54.2167, Lon: 55.0333}},
	"Баймак":          {Slug: "баймак", Coords: SettlementCoordinates{Lat: 52.5833, Lon: 58.3167}},
	"Межгорье":        {Slug: "межгорье", Coords: SettlementCoordinates{Lat: 54.0500, Lon: 57.8167}},
	"Агидель":         {Slug: "агидель", Coords: SettlementCoordinates{Lat: 55.9000, Lon: 53.9333}},

	// 40 Районов (с райцентрами)
	"Иглино":              {Slug: "иглино", Coords: SettlementCoordinates{Lat: 54.8333, Lon: 56.5000}},
	"Чишмы":               {Slug: "чишмы", Coords: SettlementCoordinates{Lat: 54.6000, Lon: 55.4000}},
	"Раевский":            {Slug: "раевский", Coords: SettlementCoordinates{Lat: 54.0667, Lon: 55.0667}},
	"Чекмагуш":            {Slug: "чекмагуш", Coords: SettlementCoordinates{Lat: 55.1333, Lon: 54.6500}},
	"Месягутово":          {Slug: "месягутово", Coords: SettlementCoordinates{Lat: 55.5333, Lon: 57.9833}},
	"Красноусольский":     {Slug: "красноусольский", Coords: SettlementCoordinates{Lat: 53.9333, Lon: 56.4833}},
	"Кармаскалы":          {Slug: "кармаскалы", Coords: SettlementCoordinates{Lat: 54.3667, Lon: 56.1667}},
	"Кушнаренково":        {Slug: "кушнаренково", Coords: SettlementCoordinates{Lat: 55.1000, Lon: 55.3500}},
	"Толбазы":             {Slug: "толбазы", Coords: SettlementCoordinates{Lat: 53.9833, Lon: 55.9333}},
	"Буздяк":              {Slug: "буздяк", Coords: SettlementCoordinates{Lat: 54.5667, Lon: 54.5333}},
	"Верхнеяркеево":       {Slug: "верхнеяркеево", Coords: SettlementCoordinates{Lat: 55.4500, Lon: 54.3167}},
	"Бураево":             {Slug: "бураево", Coords: SettlementCoordinates{Lat: 55.8333, Lon: 55.4000}},
	"Бакалы":              {Slug: "бакалы", Coords: SettlementCoordinates{Lat: 55.1833, Lon: 53.8000}},
	"Аскарово":            {Slug: "аскарово", Coords: SettlementCoordinates{Lat: 53.3333, Lon: 58.5000}},
	"Акъяр":               {Slug: "акъяр", Coords: SettlementCoordinates{Lat: 51.8667, Lon: 58.2167}},
	"Мраково":             {Slug: "мраково", Coords: SettlementCoordinates{Lat: 52.7167, Lon: 56.6333}},
	"Языково":             {Slug: "языково", Coords: SettlementCoordinates{Lat: 54.7000, Lon: 55.2000}},
	"Большеустьикинское":  {Slug: "большеустьикинское", Coords: SettlementCoordinates{Lat: 55.9500, Lon: 58.2667}},
	"Киргиз-Мияки":        {Slug: "киргиз-мияки", Coords: SettlementCoordinates{Lat: 53.6333, Lon: 54.8000}},
	"Исянгулово":          {Slug: "исянгулово", Coords: SettlementCoordinates{Lat: 52.2000, Lon: 56.6000}},
	"Аскино":              {Slug: "аскино", Coords: SettlementCoordinates{Lat: 56.0833, Lon: 56.5833}},
	"Верхние Татышлы":     {Slug: "верхние_татышлы", Coords: SettlementCoordinates{Lat: 56.2833, Lon: 55.8667}},
	"Верхние Киги":        {Slug: "верхние_киги", Coords: SettlementCoordinates{Lat: 55.4167, Lon: 58.6000}},
	"Николо-Березовка":    {Slug: "николо-березовка", Coords: SettlementCoordinates{Lat: 56.1333, Lon: 54.1500}},
	"Мишкино":             {Slug: "мишкино", Coords: SettlementCoordinates{Lat: 55.5333, Lon: 55.9500}},
	"Караидель":           {Slug: "караидель", Coords: SettlementCoordinates{Lat: 55.8333, Lon: 56.9167}},
	"Архангельское":       {Slug: "архангельское", Coords: SettlementCoordinates{Lat: 54.4000, Lon: 56.9333}},
	"Новобелокатай":       {Slug: "новобелокатай", Coords: SettlementCoordinates{Lat: 55.7000, Lon: 58.9667}},
	"Бижбуляк":            {Slug: "бижбуляк", Coords: SettlementCoordinates{Lat: 53.6833, Lon: 54.2667}},
	"Стерлибашево":        {Slug: "стерлибашево", Coords: SettlementCoordinates{Lat: 53.4333, Lon: 55.2500}},
	"Шаран":               {Slug: "шаран", Coords: SettlementCoordinates{Lat: 54.8167, Lon: 54.0000}},
	"Ермолаево":           {Slug: "ермолаево", Coords: SettlementCoordinates{Lat: 52.7167, Lon: 55.7667}},
	"Старобалтачево":      {Slug: "старобалтачево", Coords: SettlementCoordinates{Lat: 55.9833, Lon: 55.9667}},
	"Зилаир":              {Slug: "зилаир", Coords: SettlementCoordinates{Lat: 52.2333, Lon: 57.4333}},
	"Старосубхангулово":   {Slug: "старосубхангулово", Coords: SettlementCoordinates{Lat: 53.1000, Lon: 57.4500}},
	"Калтасы":             {Slug: "калтасы", Coords: SettlementCoordinates{Lat: 55.9667, Lon: 54.8000}},
	"Федоровка":           {Slug: "федоровка", Coords: SettlementCoordinates{Lat: 53.0833, Lon: 55.0833}},
	"Красная Горка":       {Slug: "красная_горка", Coords: SettlementCoordinates{Lat: 55.1833, Lon: 56.6667}},
	"Малояз":              {Slug: "малояз", Coords: SettlementCoordinates{Lat: 55.1833, Lon: 58.1667}},
	"Ермекеево":           {Slug: "ермекеево", Coords: SettlementCoordinates{Lat: 54.0667, Lon: 53.6833}},
}

var (
	sunriseRegex1 = regexp.MustCompile(`data-name="sunrise"[^>]*>([0-9]{2}:[0-9]{2}(?::[0-9]{2})?)`)
	sunriseRegex2 = regexp.MustCompile(`Восход солнца[^\d]*([0-9]{2}:[0-9]{2}(?::[0-9]{2})?)`)
	sunriseRegex3 = regexp.MustCompile(`<th[^>]*>Восход</th>.*?<td[^>]*>([0-9]{2}:[0-9]{2})</td>`)
)

type VoshodClient struct {
	httpClient *http.Client
	cache      map[string]string
	cacheMutex sync.RWMutex
}

func NewVoshodClient() *VoshodClient {
	return &VoshodClient{
		httpClient: &http.Client{Timeout: 7 * time.Second},
		cache:      make(map[string]string),
	}
}

// GetSunriseTime возвращает время восхода солнца в формате "HH:MM" для указанного города и даты
func (v *VoshodClient) GetSunriseTime(cityName string, date time.Time) (string, error) {
	cleanCity := cleanCityName(cityName)
	cacheKey := fmt.Sprintf("%s_%s", cleanCity, date.Format("2006-01-02"))

	v.cacheMutex.RLock()
	if cached, ok := v.cache[cacheKey]; ok && cached != "" {
		v.cacheMutex.RUnlock()
		return cached, nil
	}
	v.cacheMutex.RUnlock()

	slug := getSlug(cleanCity)
	sunrise, err := v.fetchSunriseFromWeb(slug, date)
	if err != nil || sunrise == "" {
		log.Printf("Предупреждение: не удалось получить восход с сайта для %s (slug: %s): %v. Используется астрономический расчет.", cleanCity, slug, err)
		sunrise = calculateAstronomicalSunrise(cleanCity, date)
	}

	if sunrise != "" {
		v.cacheMutex.Lock()
		v.cache[cacheKey] = sunrise
		v.cacheMutex.Unlock()
	}

	return sunrise, nil
}

func (v *VoshodClient) fetchSunriseFromWeb(slug string, date time.Time) (string, error) {
	escapedSlug := url.PathEscape(slug)
	reqURL := fmt.Sprintf("https://voshod-solnca.ru/sun/%s", escapedSlug)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; NamazTimeBot/1.0; +https://github.com/adexcell/prayer-time-bot)")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("статус ответа %d для URL %s", resp.StatusCode, reqURL)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	body := string(bodyBytes)

	sunrise := parseSunrise(body)
	if sunrise == "" {
		return "", fmt.Errorf("время восхода не найдено в HTML ответе")
	}

	return sunrise, nil
}

func parseSunrise(html string) string {
	if m := sunriseRegex1.FindStringSubmatch(html); len(m) > 1 {
		return trimSeconds(m[1])
	}
	if m := sunriseRegex2.FindStringSubmatch(html); len(m) > 1 {
		return trimSeconds(m[1])
	}
	if m := sunriseRegex3.FindStringSubmatch(html); len(m) > 1 {
		return trimSeconds(m[1])
	}
	return ""
}

func trimSeconds(timeStr string) string {
	parts := strings.Split(strings.TrimSpace(timeStr), ":")
	if len(parts) >= 2 {
		return fmt.Sprintf("%02s:%02s", parts[0], parts[1])
	}
	return timeStr
}

func cleanCityName(raw string) string {
	// Если передано "Иглинский р-н (Иглино)" -> "Иглино"
	if idx := strings.Index(raw, "("); idx != -1 {
		end := strings.Index(raw, ")")
		if end > idx {
			return strings.TrimSpace(raw[idx+1 : end])
		}
	}
	// Убираем префиксы р-н
	s := strings.TrimPrefix(raw, "г. ")
	s = strings.TrimPrefix(s, "р-н ")
	return strings.TrimSpace(s)
}

func getSlug(cleanCity string) string {
	if entry, ok := CitySlugsAndCoords[cleanCity]; ok && entry.Slug != "" {
		return entry.Slug
	}
	// Fallback по транслитерации/lowercase
	s := strings.ToLower(cleanCity)
	s = strings.ReplaceAll(s, " ", "_")
	return s
}

// calculateAstronomicalSunrise выполняет астрономический расчёт восхода солнца (NOAA Sunrise Equation) для UTC+5
func calculateAstronomicalSunrise(cleanCity string, date time.Time) string {
	coords := SettlementCoordinates{Lat: 54.7388, Lon: 55.9721} // По умолчанию Уфа
	if entry, ok := CitySlugsAndCoords[cleanCity]; ok {
		coords = entry.Coords
	}

	lat := coords.Lat
	lon := coords.Lon

	// День года
	dayOfYear := float64(date.YearDay())

	// 1. Приближенное время
	lngHour := lon / 15.0
	t := dayOfYear + ((6.0 - lngHour) / 24.0)

	// 2. Средняя аномалия Солнца
	M := (0.9856 * t) - 3.289

	// 3. Истинная долгота Солнца
	degToRad := math.Pi / 180.0
	radToDeg := 180.0 / math.Pi

	L := M + (1.916 * math.Sin(M*degToRad)) + (0.020 * math.Sin(2*M*degToRad)) + 282.634
	L = math.Mod(L+360.0, 360.0)

	// 4. Прямое восхождение Солнца
	RA := radToDeg * math.Atan(0.91764*math.Tan(L*degToRad))
	RA = math.Mod(RA+360.0, 360.0)

	// Коррекция квадранта RA
	Lquadrant := math.Floor(L/90.0) * 90.0
	RAquadrant := math.Floor(RA/90.0) * 90.0
	RA = RA + (Lquadrant - RAquadrant)
	RA = RA / 15.0

	// 5. Склонение Солнца
	sinDec := 0.39782 * math.Sin(L*degToRad)
	cosDec := math.Cos(math.Asin(sinDec))

	// 6. Часовой угол Солнца для восхода (зенит 90°50' = 90.8333°)
	zenith := 90.8333
	cosH := (math.Cos(zenith*degToRad) - (sinDec * math.Sin(lat*degToRad))) / (cosDec * math.Cos(lat*degToRad))

	if cosH > 1.0 || cosH < -1.0 {
		return "06:30" // Полярный день / ночь
	}

	H := 360.0 - (radToDeg * math.Acos(cosH))
	H = H / 15.0

	// 7. Местное время события
	T := H + RA - (0.06571 * t) - 6.622

	// 8. Коррекция под UTC+5 (Башкортостан)
	UT := math.Mod(T-lngHour+24.0, 24.0)
	localHour := math.Mod(UT+5.0, 24.0)

	hour := int(localHour)
	minute := int(math.Round((localHour - float64(hour)) * 60.0))
	if minute >= 60 {
		minute = 0
		hour = (hour + 1) % 24
	}

	return fmt.Sprintf("%02d:%02d", hour, minute)
}
