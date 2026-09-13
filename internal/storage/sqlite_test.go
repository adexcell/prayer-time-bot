package storage

import (
	"os"
	"testing"
)

func TestStorage_CityAndSettings(t *testing.T) {
	dbPath := "test_bot.db"
	defer os.Remove(dbPath)

	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	chatID := int64(123456)

	// 1. Get default city
	city, err := store.GetUserCity(chatID, "Уфа")
	if err != nil {
		t.Fatalf("GetUserCity failed: %v", err)
	}
	if city != "Уфа" {
		t.Errorf("Expected default city 'Уфа', got '%s'", city)
	}

	// 2. Set user city to Sterlitamak
	if err := store.SetUserCity(chatID, "Стерлитамак"); err != nil {
		t.Fatalf("SetUserCity failed: %v", err)
	}

	city, err = store.GetUserCity(chatID, "Уфа")
	if err != nil {
		t.Fatalf("GetUserCity failed: %v", err)
	}
	if city != "Стерлитамак" {
		t.Errorf("Expected city 'Стерлитамак', got '%s'", city)
	}

	// 3. Update daily time to 07:00
	if err := store.UpdateDailyTime(chatID, "07:00"); err != nil {
		t.Fatalf("UpdateDailyTime failed: %v", err)
	}

	u, err := store.GetUser(chatID)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if u.DailyScheduleTime != "07:00" {
		t.Errorf("Expected DailyScheduleTime '07:00', got '%s'", u.DailyScheduleTime)
	}

	// 4. Toggle notify 15min
	if err := store.ToggleNotify15min(chatID); err != nil {
		t.Fatalf("ToggleNotify15min failed: %v", err)
	}
	u, _ = store.GetUser(chatID)
	if u.Notify15Min != false {
		t.Errorf("Expected Notify15Min false after toggle, got %v", u.Notify15Min)
	}
}
