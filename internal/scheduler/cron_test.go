package scheduler

import (
	"os"
	"testing"
	"time"

	"namaz-time-bot/internal/storage"
)

func TestGetPrayerHadithCategory(t *testing.T) {
	if cat := getPrayerHadithCategory("Фаджр", time.Monday); cat != storage.CategoryPrayerFajr {
		t.Errorf("Expected CategoryPrayerFajr, got %s", cat)
	}
	if cat := getPrayerHadithCategory("Зухр", time.Friday); cat != storage.CategoryPrayerJumah {
		t.Errorf("Expected CategoryPrayerJumah on Friday, got %s", cat)
	}
	if cat := getPrayerHadithCategory("Зухр", time.Wednesday); cat != storage.CategoryPrayerDhuhr {
		t.Errorf("Expected CategoryPrayerDhuhr on Wednesday, got %s", cat)
	}
	if cat := getPrayerHadithCategory("Аср", time.Tuesday); cat != storage.CategoryPrayerAsr {
		t.Errorf("Expected CategoryPrayerAsr, got %s", cat)
	}
	if cat := getPrayerHadithCategory("Магриб", time.Saturday); cat != storage.CategoryPrayerMaghrib {
		t.Errorf("Expected CategoryPrayerMaghrib, got %s", cat)
	}
	if cat := getPrayerHadithCategory("Иша", time.Sunday); cat != storage.CategoryPrayerIsha {
		t.Errorf("Expected CategoryPrayerIsha, got %s", cat)
	}
}

func TestScheduler_HadithExecution(t *testing.T) {
	dbPath := "test_scheduler_hadiths.db"
	defer os.Remove(dbPath)

	store, err := storage.New(dbPath)
	if err != nil {
		t.Fatalf("Storage creation failed: %v", err)
	}
	defer store.Close()

	// Verify that initial hadiths exist
	h, err := store.GetRandomHadith(storage.CategoryFastingWhiteDays)
	if err != nil || h == nil {
		t.Fatalf("Expected fasting white days hadith, got err=%v, h=%+v", err, h)
	}
	if h.Source == "" || h.Text == "" {
		t.Errorf("Hadith missing text or source: %+v", h)
	}

	hPray, err := store.GetRandomHadith(storage.CategoryPrayerFajr)
	if err != nil || hPray == nil {
		t.Fatalf("Expected prayer fajr hadith, got err=%v, h=%+v", err, hPray)
	}
	if hPray.Source == "" || hPray.Text == "" {
		t.Errorf("Prayer hadith missing text or source: %+v", hPray)
	}
}
