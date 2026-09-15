package api

import (
	"testing"
	"time"
)

func TestParseSunrise(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "today-list markup with seconds",
			html:     `<div data-name="sunrise" class="today-list__item-value font-weight-bold" data-timestamp="1789436876">06:48:56</div>`,
			expected: "06:48",
		},
		{
			name:     "today-list markup HH:MM",
			html:     `<div data-name="sunrise" class="today-list__item-value">06:50</div>`,
			expected: "06:50",
		},
		{
			name:     "textual fallback markup",
			html:     `<div>Восход солнца 06:47:12</div>`,
			expected: "06:47",
		},
		{
			name:     "table markup",
			html:     `<th>Восход</th><td>06:52</td>`,
			expected: "06:52",
		},
		{
			name:     "empty html",
			html:     `<div>No sunrise data</div>`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSunrise(tt.html)
			if got != tt.expected {
				t.Errorf("parseSunrise() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestCleanCityName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Уфа", "Уфа"},
		{"г. Стерлитамак", "Стерлитамак"},
		{"Иглинский р-н (Иглино)", "Иглино"},
		{"Чишминский р-н (Чишмы)", "Чишмы"},
		{"р-н Буздяк", "Буздяк"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := cleanCityName(tt.input)
			if got != tt.expected {
				t.Errorf("cleanCityName(%q) = %q, expected %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGetSlug(t *testing.T) {
	tests := []struct {
		city     string
		expected string
	}{
		{"Уфа", "уфа"},
		{"Стерлитамак", "стерлитамак"},
		{"Благовещенск", "благовещенск_(башкортостан)"},
		{"Октябрьский", "октябрьский_(башкортостан)"},
		{"Верхние Татышлы", "верхние_татышлы"},
		{"Иглино", "иглино"},
	}

	for _, tt := range tests {
		t.Run(tt.city, func(t *testing.T) {
			got := getSlug(tt.city)
			if got != tt.expected {
				t.Errorf("getSlug(%q) = %q, expected %q", tt.city, got, tt.expected)
			}
		})
	}
}

func TestCalculateAstronomicalSunrise(t *testing.T) {
	// 15 сентября 2026 для Уфы
	date := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	sunrise := calculateAstronomicalSunrise("Уфа", date)

	if sunrise == "" || len(sunrise) != 5 {
		t.Fatalf("calculateAstronomicalSunrise() returned invalid format: %q", sunrise)
	}

	// В сентябре для Уфы восход около 06:40-06:55
	if sunrise < "06:30" || sunrise > "07:05" {
		t.Errorf("calculateAstronomicalSunrise(Уфа, 15.09) = %q, expected between 06:30 and 07:05", sunrise)
	}
}

func TestVoshodClientCache(t *testing.T) {
	client := NewVoshodClient()

	date := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	// Добавим в кэш
	cacheKey := "Уфа_2026-09-15"
	client.cache[cacheKey] = "06:48"

	got, err := client.GetSunriseTime("Уфа", date)
	if err != nil {
		t.Fatalf("GetSunriseTime failed: %v", err)
	}
	if got != "06:48" {
		t.Errorf("GetSunriseTime() from cache = %q, expected '06:48'", got)
	}
}
