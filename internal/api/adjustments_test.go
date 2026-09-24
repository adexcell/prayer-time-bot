package api

import (
	"strings"
	"testing"
	"time"
)

func TestApplyOffset(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		offset   int
		expected string
	}{
		{"positive offset", "19:30", 5, "19:35"},
		{"negative offset", "04:40", -5, "04:35"},
		{"zero offset", "13:15", 0, "13:15"},
		{"wrap forward over midnight", "23:58", 5, "00:03"},
		{"wrap backward over midnight", "00:02", -5, "23:57"},
		{"large offset", "12:00", 90, "13:30"},
		{"invalid format returns original", "invalid", 5, "invalid"},
		{"empty string returns empty", "", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyOffset(tt.input, tt.offset)
			if got != tt.expected {
				t.Errorf("ApplyOffset(%q, %d) = %q, expected %q", tt.input, tt.offset, got, tt.expected)
			}
		})
	}
}

func TestApplyPrayerAdjustments(t *testing.T) {
	item := &DUMRBItem{
		SuhurDo: "04:38",
		Fajr:    "04:53",
		Sunrise: "06:48",
		Dhuhr:   "13:11",
		Asr:     "16:35",
		Maghrib: "19:33",
		Isha:    "21:32",
	}

	adjustments := map[string]PrayerRule{
		"Иша":   {OffsetMinutes: 5},  // 21:32 -> 21:37
		"Фаджр": {OffsetMinutes: -3}, // 04:53 -> 04:50, SuhurDo 04:38 -> 04:35
	}

	ApplyPrayerAdjustments(item, adjustments)

	if item.Isha != "21:37" {
		t.Errorf("Expected Isha '21:37', got %q", item.Isha)
	}
	if item.Fajr != "04:50" {
		t.Errorf("Expected Fajr '04:50', got %q", item.Fajr)
	}
	if item.SuhurDo != "04:35" {
		t.Errorf("Expected SuhurDo '04:35', got %q", item.SuhurDo)
	}
	if item.Dhuhr != "13:11" {
		t.Errorf("Expected unchanged Dhuhr '13:11', got %q", item.Dhuhr)
	}
}

func TestApplyPrayerAdjustments_FixedTime(t *testing.T) {
	item := &DUMRBItem{
		SuhurDo: "04:38",
		Fajr:    "04:53",
		Sunrise: "06:48",
		Dhuhr:   "13:11",
		Asr:     "16:35",
		Maghrib: "19:33",
		Isha:    "21:32",
	}

	adjustments := map[string]PrayerRule{
		"Зухр":  {FixedTime: "13:30"},
		"Фаджр": {FixedTime: "05:00"},
	}

	ApplyPrayerAdjustments(item, adjustments)

	if item.Dhuhr != "13:30" {
		t.Errorf("Expected fixed Dhuhr '13:30', got %q", item.Dhuhr)
	}
	if item.Fajr != "05:00" {
		t.Errorf("Expected fixed Fajr '05:00', got %q", item.Fajr)
	}
	if item.SuhurDo != "05:00" {
		t.Errorf("Expected SuhurDo '05:00', got %q", item.SuhurDo)
	}
	if item.Isha != "21:32" {
		t.Errorf("Expected unchanged Isha '21:32', got %q", item.Isha)
	}
}

func TestApplyPrayerAdjustments_All(t *testing.T) {
	item := &DUMRBItem{
		Fajr:    "05:00",
		Sunrise: "06:00",
		Dhuhr:   "13:00",
		Asr:     "16:00",
		Maghrib: "19:00",
		Isha:    "21:00",
	}

	adjustments := map[string]PrayerRule{
		"all": {OffsetMinutes: 2},
	}

	ApplyPrayerAdjustments(item, adjustments)

	if item.Fajr != "05:02" || item.Sunrise != "06:02" || item.Dhuhr != "13:02" ||
		item.Asr != "16:02" || item.Maghrib != "19:02" || item.Isha != "21:02" {
		t.Errorf("Unexpected result with 'all' adjustment: %+v", item)
	}
}

func TestFormatMessageCustom(t *testing.T) {
	item := &DUMRBItem{
		Fajr:    "05:00",
		Sunrise: "06:30",
		Dhuhr:   "13:00",
		Asr:     "16:30",
		Maghrib: "19:00",
		Isha:    "21:00",
	}
	date := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)

	// 1. Default (Russian)
	msgRu := FormatMessage(item, "Уфа", date)
	if !strings.Contains(msgRu, "Фаджр") || !strings.Contains(msgRu, "05:00") {
		t.Errorf("Expected Russian format to contain 'Фаджр', got: %s", msgRu)
	}

	// 2. Arabic preset
	msgAr := FormatMessageCustom(item, "Уфа", date, PrayerFormatConfig{
		Preset:       "ar",
		CustomHeader: "🕌 *مواقيت الصلاة في أوفا*",
		CustomFooter: "📢 @mychannel",
	})
	if !strings.Contains(msgAr, "الفجر") || !strings.Contains(msgAr, "مواقيت الصلاة في أوفا") || !strings.Contains(msgAr, "@mychannel") {
		t.Errorf("Expected Arabic format with custom header and footer, got: %s", msgAr)
	}

	// 3. Bilingual preset
	msgBi := FormatMessageCustom(item, "Уфа", date, PrayerFormatConfig{
		Preset: "ru_ar",
	})
	if !strings.Contains(msgBi, "Фаджр (الفجر)") {
		t.Errorf("Expected Bilingual format with 'Фаджр (الفجر)', got: %s", msgBi)
	}

	// 4. Bashkir preset
	msgBa := FormatMessageCustom(item, "Уфа", date, PrayerFormatConfig{
		Preset: "ba",
	})
	if !strings.Contains(msgBa, "Иртәнге") || !strings.Contains(msgBa, "Кояш сығыуы") {
		t.Errorf("Expected Bashkir format with 'Иртәнге', got: %s", msgBa)
	}

	// 5. Custom footer with Markdown hyperlink
	msgLink := FormatMessageCustom(item, "Уфа", date, PrayerFormatConfig{
		Preset:       "ru",
		CustomFooter: "📢 [Наш канал](https://t.me/mychannel)",
	})
	if !strings.Contains(msgLink, "📢 [Наш канал](https://t.me/mychannel)") {
		t.Errorf("Expected message to contain custom markdown link footer, got: %s", msgLink)
	}
}

func TestFormatEveningMessageCustom(t *testing.T) {
	today := &DUMRBItem{
		Maghrib: "19:00",
		Isha:    "21:00",
	}
	tomorrow := &DUMRBItem{
		Fajr:    "05:01",
		Sunrise: "06:31",
		Dhuhr:   "13:00",
		Asr:     "16:29",
		Maghrib: "18:58",
		Isha:    "20:58",
	}
	date := time.Date(2026, 9, 21, 20, 0, 0, 0, time.UTC)

	msg := FormatEveningMessageCustom(today, tomorrow, "Уфа", date, PrayerFormatConfig{
		Preset:       "ar",
		CustomFooter: "📢 @channel",
	})

	if !strings.Contains(msg, "المغرب") || !strings.Contains(msg, "@channel") {
		t.Errorf("Expected evening arabic format, got: %s", msg)
	}
}

func TestFormatMessageCustom_IndividualPrayerOverridesWithEmoji(t *testing.T) {
	item := &DUMRBItem{
		Fajr:    "05:00",
		Sunrise: "06:30",
		Dhuhr:   "13:00",
		Asr:     "16:30",
		Maghrib: "19:00",
		Isha:    "21:00",
	}
	date := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)

	cfg := PrayerFormatConfig{
		Preset: "ru",
		CustomPrayers: map[string]string{
			"fajr":    "✨ 🌅 Фаджр Намаз",
			"maghrib": "🌙 Аҡшам (Магриб)",
		},
	}

	msg := FormatMessageCustom(item, "Уфа", date, cfg)

	if !strings.Contains(msg, "✨ 🌅 Фаджр Намаз") {
		t.Errorf("Expected custom Fajr override with emoji, got: %s", msg)
	}
	if !strings.Contains(msg, "🌙 Аҡшам (Магриб)") {
		t.Errorf("Expected custom Maghrib override, got: %s", msg)
	}
	// Other prayers should remain default Russian
	if !strings.Contains(msg, "Зухр") {
		t.Errorf("Expected default Dhuhr in Russian, got: %s", msg)
	}
}
