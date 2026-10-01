package storage

import (
	"os"
	"testing"
)

func TestStorage_Hadiths(t *testing.T) {
	dbPath := "test_hadiths.db"
	defer os.Remove(dbPath)

	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	// 1. Check seeded hadiths count
	total, active, byCat, byCol, err := store.GetHadithsStats()
	if err != nil {
		t.Fatalf("GetHadithsStats failed: %v", err)
	}
	if total == 0 || active == 0 {
		t.Errorf("Expected seeded hadiths, got total=%d, active=%d", total, active)
	}
	if len(byCat) == 0 || len(byCol) == 0 {
		t.Errorf("Expected populated byCat and byCol, got %v and %v", byCat, byCol)
	}

	// 2. Add custom hadith
	id, err := store.AddHadith("Тестовый текст хадиса", "Сахих аль-Бухари, 123", CollectionBukhari, CategoryGeneral)
	if err != nil {
		t.Fatalf("AddHadith failed: %v", err)
	}
	if id <= 0 {
		t.Errorf("Expected valid hadith id, got %d", id)
	}

	h, err := store.GetHadithByID(id)
	if err != nil {
		t.Fatalf("GetHadithByID failed: %v", err)
	}
	if h == nil || h.Text != "Тестовый текст хадиса" || h.Source != "Сахих аль-Бухари, 123" {
		t.Errorf("Unexpected hadith returned: %+v", h)
	}

	// 3. Toggle active
	if err := store.ToggleHadithActive(id); err != nil {
		t.Fatalf("ToggleHadithActive failed: %v", err)
	}
	h, _ = store.GetHadithByID(id)
	if h.IsActive != false {
		t.Errorf("Expected IsActive to be false, got %v", h.IsActive)
	}

	// 4. Random hadith
	rndH, err := store.GetRandomHadith(CategoryPrayerFajr)
	if err != nil {
		t.Fatalf("GetRandomHadith failed: %v", err)
	}
	if rndH == nil || rndH.Category != CategoryPrayerFajr {
		t.Errorf("Expected Fajr hadith, got %+v", rndH)
	}

	// 5. Delete hadith
	if err := store.DeleteHadith(id); err != nil {
		t.Fatalf("DeleteHadith failed: %v", err)
	}
	h, err = store.GetHadithByID(id)
	if err != nil || h != nil {
		t.Errorf("Expected hadith to be deleted, got err=%v, h=%+v", err, h)
	}
}
