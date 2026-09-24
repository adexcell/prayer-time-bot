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

	// 5. Update evening time to 19:00
	if err := store.UpdateEveningTime(chatID, "19:00"); err != nil {
		t.Fatalf("UpdateEveningTime failed: %v", err)
	}
	u, _ = store.GetUser(chatID)
	if u.EveningScheduleTime != "19:00" {
		t.Errorf("Expected EveningScheduleTime '19:00', got '%s'", u.EveningScheduleTime)
	}

	// 6. Test normalization of "10" and "21"
	if err := store.UpdateDailyTime(chatID, "10"); err != nil {
		t.Fatalf("UpdateDailyTime with '10' failed: %v", err)
	}
	if err := store.UpdateEveningTime(chatID, "21"); err != nil {
		t.Fatalf("UpdateEveningTime with '21' failed: %v", err)
	}
	u, _ = store.GetUser(chatID)
	if u.DailyScheduleTime != "10:00" {
		t.Errorf("Expected normalized DailyScheduleTime '10:00', got '%s'", u.DailyScheduleTime)
	}
	if u.EveningScheduleTime != "21:00" {
		t.Errorf("Expected normalized EveningScheduleTime '21:00', got '%s'", u.EveningScheduleTime)
	}
}

func TestStorage_ManagedChatsAndCustomization(t *testing.T) {
	dbPath := "test_chats.db"
	defer os.Remove(dbPath)

	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	channel1 := int64(-100111)
	channel2 := int64(-100222)
	adminID := int64(999)

	// 1. Save chat info
	if err := store.SaveChatInfo(channel1, "Канал 1", "channel", adminID); err != nil {
		t.Fatalf("SaveChatInfo failed: %v", err)
	}
	if err := store.SaveChatInfo(channel2, "Канал 2", "channel", adminID); err != nil {
		t.Fatalf("SaveChatInfo failed: %v", err)
	}
	_ = store.SetUserCity(channel1, "Уфа")
	_ = store.SetUserCity(channel2, "Стерлитамак")

	// 2. Retrieve managed chats
	chats, err := store.GetManagedChats(adminID, false)
	if err != nil || len(chats) != 2 {
		t.Fatalf("Expected 2 managed chats, got %d (err: %v)", len(chats), err)
	}

	// 3. Customization
	if err := store.UpdateChatPreset(channel1, "ar"); err != nil {
		t.Fatalf("UpdateChatPreset failed: %v", err)
	}
	if err := store.UpdateChatFooter(channel1, "📢 @channel1"); err != nil {
		t.Fatalf("UpdateChatFooter failed: %v", err)
	}
	if err := store.UpdateChatHeader(channel1, "🕌 Расписание"); err != nil {
		t.Fatalf("UpdateChatHeader failed: %v", err)
	}
	if err := store.UpdateCustomPrayerName(channel1, "fajr", "✨ 🌅 Фаджр (Кастом)"); err != nil {
		t.Fatalf("UpdateCustomPrayerName failed: %v", err)
	}

	u1, err := store.GetUser(channel1)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if u1.PrayerNamesPreset != "ar" || u1.CustomFooter != "📢 @channel1" || u1.CustomHeader != "🕌 Расписание" {
		t.Errorf("Unexpected customization values: %+v", u1)
	}
	pMap := ParseCustomPrayerNames(u1.CustomPrayerNames)
	if pMap["fajr"] != "✨ 🌅 Фаджр (Кастом)" {
		t.Errorf("Expected custom Fajr name, got %v", pMap)
	}

	// 4. Remove single prayer override with '-'
	if err := store.UpdateCustomPrayerName(channel1, "fajr", "-"); err != nil {
		t.Fatalf("UpdateCustomPrayerName with '-' failed: %v", err)
	}
	u1, _ = store.GetUser(channel1)
	pMap = ParseCustomPrayerNames(u1.CustomPrayerNames)
	if _, exists := pMap["fajr"]; exists {
		t.Errorf("Expected fajr override to be deleted, got %v", pMap)
	}

	// 5. Reset customization
	if err := store.ResetChatCustomization(channel1); err != nil {
		t.Fatalf("ResetChatCustomization failed: %v", err)
	}
	u1, _ = store.GetUser(channel1)
	if u1.PrayerNamesPreset != "ru" || u1.CustomFooter != "" || u1.CustomHeader != "" || u1.CustomPrayerNames != "" {
		t.Errorf("Expected reset customization values, got %+v", u1)
	}

	// 6. Delete chat
	if err := store.DeleteChat(channel1); err != nil {
		t.Fatalf("DeleteChat failed: %v", err)
	}
	chatsAfter, err := store.GetManagedChats(adminID, false)
	if err != nil || len(chatsAfter) != 1 {
		t.Fatalf("Expected 1 chat after deletion, got %d", len(chatsAfter))
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

func TestNormalizeBroadcastTime(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"06:00", "06:00", false},
		{"6:00", "06:00", false},
		{"6.30", "06:30", false},
		{"19-45", "19:45", false},
		{"21 00", "21:00", false},
		{"7", "07:00", false},
		{"00:00", "00:00", false},
		{"23:59", "23:59", false},
		{"24:00", "", true},
		{"12:60", "", true},
		{"abc", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		got, err := NormalizeBroadcastTime(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("NormalizeBroadcastTime(%q) expected error, got %q", tt.input, got)
			}
		} else {
			if err != nil || got != tt.expected {
				t.Errorf("NormalizeBroadcastTime(%q) = %q (err: %v), expected %q", tt.input, got, err, tt.expected)
			}
		}
	}
}

func TestStorage_BroadcastTimes(t *testing.T) {
	dbPath := "test_broadcast_times.db"
	defer os.Remove(dbPath)

	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	chatID := int64(777888)

	// 1. Initial user has default ["06:00"]
	_ = store.Subscribe(chatID, "Уфа")
	u, err := store.GetUser(chatID)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	times := u.GetParsedBroadcastTimes()
	if len(times) != 1 || times[0] != "06:00" {
		t.Errorf("Expected default time ['06:00'], got %v", times)
	}

	// 2. Add second time 19:30
	norm, err := store.AddBroadcastTime(chatID, "19:30")
	if err != nil || norm != "19:30" {
		t.Fatalf("AddBroadcastTime failed: %v", err)
	}

	// 3. Add third time 12:00
	_, _ = store.AddBroadcastTime(chatID, "12:00")

	u, _ = store.GetUser(chatID)
	times = u.GetParsedBroadcastTimes()
	if len(times) != 3 || times[0] != "06:00" || times[1] != "12:00" || times[2] != "19:30" {
		t.Errorf("Expected sorted times ['06:00', '12:00', '19:30'], got %v", times)
	}

	// 4. Remove time 12:00
	if err := store.RemoveBroadcastTime(chatID, "12:00"); err != nil {
		t.Fatalf("RemoveBroadcastTime failed: %v", err)
	}
	u, _ = store.GetUser(chatID)
	times = u.GetParsedBroadcastTimes()
	if len(times) != 2 || times[0] != "06:00" || times[1] != "19:30" {
		t.Errorf("Expected times ['06:00', '19:30'] after removal, got %v", times)
	}

	// 5. Clear all broadcast times
	if err := store.ClearBroadcastTimes(chatID); err != nil {
		t.Fatalf("ClearBroadcastTimes failed: %v", err)
	}
	u, _ = store.GetUser(chatID)
	times = u.GetParsedBroadcastTimes()
	if len(times) != 0 {
		t.Errorf("Expected 0 times after clear, got %v", times)
	}
}
