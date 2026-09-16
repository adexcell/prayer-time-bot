package storage

import (
	"os"
	"testing"
	"time"
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

func TestStorage_Admins(t *testing.T) {
	dbPath := "test_admins.db"
	defer os.Remove(dbPath)

	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	configAdmins := []int64{111, 222}

	// 1. Config admin check
	if !store.IsAdmin(111, configAdmins) {
		t.Errorf("Expected user 111 to be admin from config")
	}
	if store.IsAdmin(333, configAdmins) {
		t.Errorf("Expected user 333 NOT to be admin")
	}

	// 2. Add dynamic admin
	if err := store.AddAdmin(333, "tester", 111); err != nil {
		t.Fatalf("AddAdmin failed: %v", err)
	}
	if !store.IsAdmin(333, configAdmins) {
		t.Errorf("Expected user 333 to be admin after AddAdmin")
	}

	admins, err := store.GetAdmins()
	if err != nil || len(admins) != 1 {
		t.Fatalf("GetAdmins failed or returned wrong count: %v, len: %d", err, len(admins))
	}
	if admins[0].UserID != 333 || admins[0].Username != "tester" {
		t.Errorf("GetAdmins returned wrong admin data: %+v", admins[0])
	}

	// 3. Remove admin
	if err := store.RemoveAdmin(333); err != nil {
		t.Fatalf("RemoveAdmin failed: %v", err)
	}
	if store.IsAdmin(333, configAdmins) {
		t.Errorf("Expected user 333 NOT to be admin after RemoveAdmin")
	}
}

func TestStorage_PrayerAdjustments(t *testing.T) {
	dbPath := "test_adjustments.db"
	defer os.Remove(dbPath)

	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	now := time.Now()
	validUntil := now.Add(7 * 24 * time.Hour)

	// 1. Save adjustment: Ufa, Isha +5
	if err := store.SaveAdjustment("Уфа", "Иша", 5, validUntil); err != nil {
		t.Fatalf("SaveAdjustment failed: %v", err)
	}
	if err := store.SaveAdjustment("Уфа", "Фаджр", -2, validUntil); err != nil {
		t.Fatalf("SaveAdjustment failed: %v", err)
	}

	// 2. Get active adjustments for today
	adj, err := store.GetActiveAdjustments("Уфа", now)
	if err != nil {
		t.Fatalf("GetActiveAdjustments failed: %v", err)
	}
	if adj["Иша"] != 5 || adj["Фаджр"] != -2 {
		t.Errorf("Unexpected adjustments for Ufa: %+v", adj)
	}

	// 3. For another city should be empty
	adjSterlitamak, err := store.GetActiveAdjustments("Стерлитамак", now)
	if err != nil || len(adjSterlitamak) != 0 {
		t.Errorf("Expected 0 adjustments for Sterlitamak, got %+v", adjSterlitamak)
	}

	// 4. GetAllActiveAdjustments
	all, err := store.GetAllActiveAdjustments()
	if err != nil || len(all) != 2 {
		t.Fatalf("GetAllActiveAdjustments returned wrong count: %d", len(all))
	}

	// 5. Delete adjustment
	if err := store.DeleteAdjustment(all[0].ID); err != nil {
		t.Fatalf("DeleteAdjustment failed: %v", err)
	}
	allAfter, _ := store.GetAllActiveAdjustments()
	if len(allAfter) != 1 {
		t.Errorf("Expected 1 adjustment left after deletion, got %d", len(allAfter))
	}
}
