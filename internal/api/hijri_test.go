package api

import (
	"strings"
	"testing"
	"time"
)

func TestHijriCalculationAndLocalization(t *testing.T) {
	// 29 сентября 2026 года -> 18 Раби ас-сани 1448
	date := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

	SetGlobalHijriOffset(0)
	hd, err := GetHijriDate(date, 0)
	if err != nil {
		t.Fatalf("Unexpected error calculating Hijri date: %v", err)
	}

	if hd.Day != 18 || hd.Month != 4 || hd.Year != 1448 {
		t.Errorf("Expected 18-04-1448, got: %d-%d-%d", hd.Day, hd.Month, hd.Year)
	}

	// Test Russian formatting
	ruStr := FormatHijriDate(date, "ru")
	if ruStr != "18 Раби ас-сани 1448 г. х." {
		t.Errorf("Expected Russian format '18 Раби ас-сани 1448 г. х.', got: %s", ruStr)
	}

	// Test Bashkir formatting
	baStr := FormatHijriDate(date, "ba")
	if baStr != "18 Рабиғел-ахыр 1448 һ. й." {
		t.Errorf("Expected Bashkir format '18 Рабиғел-ахыр 1448 һ. й.', got: %s", baStr)
	}

	// Test Arabic formatting
	arStr := FormatHijriDate(date, "ar")
	if arStr != "18 ربيع الثاني 1448 هـ" {
		t.Errorf("Expected Arabic format '18 ربيع الثاني 1448 هـ', got: %s", arStr)
	}

	// Test Bilingual formatting
	biStr := FormatHijriDate(date, "ru_ar")
	if biStr != "18 Раби ас-сани (ربيع الثاني) 1448 г. х." {
		t.Errorf("Expected Bilingual format '18 Раби ас-сани (ربيع الثاني) 1448 г. х.', got: %s", biStr)
	}
}

func TestHijriOffset(t *testing.T) {
	date := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

	// Base is day 18
	// With offset +1, it should be day 19
	hdPlus, err := GetHijriDate(date, 1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if hdPlus.Day != 19 {
		t.Errorf("Expected day 19 with +1 offset, got: %d", hdPlus.Day)
	}

	// With offset -1, it should be day 17
	hdMinus, err := GetHijriDate(date, -1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if hdMinus.Day != 17 {
		t.Errorf("Expected day 17 with -1 offset, got: %d", hdMinus.Day)
	}

	// Test GlobalHijriOffset
	SetGlobalHijriOffset(1)
	if GetGlobalHijriOffset() != 1 {
		t.Errorf("Expected global offset 1, got: %d", GetGlobalHijriOffset())
	}
	formattedPlus := FormatHijriDate(date, "ru")
	if !strings.HasPrefix(formattedPlus, "19 ") {
		t.Errorf("Expected formatted string with global offset to start with '19 ', got: %s", formattedPlus)
	}

	// Reset global offset
	SetGlobalHijriOffset(0)
}

func TestFindOffsetForTargetDay(t *testing.T) {
	// Date: 2026-09-29 -> base day is 18
	date := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

	// If clergy decided today is 19th -> offset should be +1
	offset, err := FindOffsetForTargetDay(date, 19)
	if err != nil {
		t.Fatalf("FindOffsetForTargetDay failed: %v", err)
	}
	if offset != 1 {
		t.Errorf("Expected offset 1 for day 19, got: %d", offset)
	}

	// If clergy decided today is 17th -> offset should be -1
	offset, err = FindOffsetForTargetDay(date, 17)
	if err != nil {
		t.Fatalf("FindOffsetForTargetDay failed: %v", err)
	}
	if offset != -1 {
		t.Errorf("Expected offset -1 for day 17, got: %d", offset)
	}

	// If target day is out of reach (> 5 days diff)
	_, err = FindOffsetForTargetDay(date, 5)
	if err == nil {
		t.Errorf("Expected error for unreachable target day, got nil")
	}
}

func TestFormatMessageContainsHijri(t *testing.T) {
	item := &DUMRBItem{
		Fajr:    "05:00",
		Sunrise: "06:30",
		Dhuhr:   "13:00",
		Asr:     "16:30",
		Maghrib: "19:00",
		Isha:    "21:00",
	}
	date := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	SetGlobalHijriOffset(0)

	msg := FormatMessage(item, "Уфа", date)
	if !strings.Contains(msg, "18 Раби ас-сани 1448 г. х.") {
		t.Errorf("Expected FormatMessage to include Hijri date, got: %s", msg)
	}
}
