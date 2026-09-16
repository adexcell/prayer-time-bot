package api

import (
	"testing"
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

	adjustments := map[string]int{
		"Иша":   5,  // 21:32 -> 21:37
		"Фаджр": -3, // 04:53 -> 04:50, SuhurDo 04:38 -> 04:35
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

func TestApplyPrayerAdjustments_All(t *testing.T) {
	item := &DUMRBItem{
		Fajr:    "05:00",
		Sunrise: "06:00",
		Dhuhr:   "13:00",
		Asr:     "16:00",
		Maghrib: "19:00",
		Isha:    "21:00",
	}

	adjustments := map[string]int{
		"all": 2,
	}

	ApplyPrayerAdjustments(item, adjustments)

	if item.Fajr != "05:02" || item.Sunrise != "06:02" || item.Dhuhr != "13:02" ||
		item.Asr != "16:02" || item.Maghrib != "19:02" || item.Isha != "21:02" {
		t.Errorf("Unexpected result with 'all' adjustment: %+v", item)
	}
}
