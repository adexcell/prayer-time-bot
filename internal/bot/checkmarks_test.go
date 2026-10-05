package bot

import (
	"os"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"namaz-time-bot/internal/storage"
)

func newTestBot(t *testing.T, dbPath string) (*Bot, *storage.Storage) {
	_ = os.Remove(dbPath)
	store, err := storage.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	b := &Bot{
		storage:        store,
		configAdminIDs: []int64{1001},
		defaultCity:    "Уфа",
	}
	return b, store
}

func makeCallback(fromID, chatID int64, data string) *tgbotapi.CallbackQuery {
	return &tgbotapi.CallbackQuery{
		ID:   "cb_test",
		Data: data,
		From: &tgbotapi.User{
			ID: fromID,
		},
		Message: &tgbotapi.Message{
			MessageID: 100,
			Chat: &tgbotapi.Chat{
				ID: chatID,
			},
		},
	}
}

// TestUserSettings_Checkmarks проверяет переключение галочек в личных настройках пользователя
func TestUserSettings_Checkmarks(t *testing.T) {
	dbPath := "test_user_settings_checkmarks.db"
	defer os.Remove(dbPath)
	b, store := newTestBot(t, dbPath)
	defer store.Close()

	userID := int64(12345)

	// 1. toggle_15min
	cb := makeCallback(userID, userID, "toggle_15min")
	b.handleCallback(cb)
	u, _ := store.GetUser(userID)
	if u.Notify15Min != false {
		t.Errorf("Expected Notify15Min false after toggle, got %v", u.Notify15Min)
	}
	b.handleCallback(cb)
	u, _ = store.GetUser(userID)
	if u.Notify15Min != true {
		t.Errorf("Expected Notify15Min true after 2nd toggle, got %v", u.Notify15Min)
	}

	// 2. toggle_attime
	cb = makeCallback(userID, userID, "toggle_attime")
	b.handleCallback(cb)
	u, _ = store.GetUser(userID)
	if u.NotifyAtTime != false {
		t.Errorf("Expected NotifyAtTime false after toggle, got %v", u.NotifyAtTime)
	}
	b.handleCallback(cb)
	u, _ = store.GetUser(userID)
	if u.NotifyAtTime != true {
		t.Errorf("Expected NotifyAtTime true after 2nd toggle, got %v", u.NotifyAtTime)
	}

	// 3. toggle_hadith_daily
	cb = makeCallback(userID, userID, "toggle_hadith_daily")
	b.handleCallback(cb)
	u, _ = store.GetUser(userID)
	if u.HadithDailyEnabled != false {
		t.Errorf("Expected HadithDailyEnabled false after toggle, got %v", u.HadithDailyEnabled)
	}
	b.handleCallback(cb)
	u, _ = store.GetUser(userID)
	if u.HadithDailyEnabled != true {
		t.Errorf("Expected HadithDailyEnabled true after 2nd toggle, got %v", u.HadithDailyEnabled)
	}

	// 4. toggle_hadith_prayer
	cb = makeCallback(userID, userID, "toggle_hadith_prayer")
	b.handleCallback(cb)
	u, _ = store.GetUser(userID)
	if u.HadithPrayerEnabled != false {
		t.Errorf("Expected HadithPrayerEnabled false after toggle, got %v", u.HadithPrayerEnabled)
	}
	b.handleCallback(cb)
	u, _ = store.GetUser(userID)
	if u.HadithPrayerEnabled != true {
		t.Errorf("Expected HadithPrayerEnabled true after 2nd toggle, got %v", u.HadithPrayerEnabled)
	}
}

// TestGroupSettings_Checkmarks проверяет переключение галочек в настройках группы
func TestGroupSettings_Checkmarks(t *testing.T) {
	dbPath := "test_group_settings_checkmarks.db"
	defer os.Remove(dbPath)
	b, store := newTestBot(t, dbPath)
	defer store.Close()

	adminID := int64(1001)
	groupID := int64(-100999)

	// 1. gtog15
	cb := makeCallback(adminID, adminID, "gtog15:"+string("-100999"))
	b.handleCallback(cb)
	u, _ := store.GetUser(groupID)
	if u.Notify15Min != false {
		t.Errorf("Expected group Notify15Min false after toggle, got %v", u.Notify15Min)
	}
	b.handleCallback(cb)
	u, _ = store.GetUser(groupID)
	if u.Notify15Min != true {
		t.Errorf("Expected group Notify15Min true after 2nd toggle, got %v", u.Notify15Min)
	}

	// 2. gtogat
	cb = makeCallback(adminID, adminID, "gtogat:"+string("-100999"))
	b.handleCallback(cb)
	u, _ = store.GetUser(groupID)
	if u.NotifyAtTime != false {
		t.Errorf("Expected group NotifyAtTime false after toggle, got %v", u.NotifyAtTime)
	}
	b.handleCallback(cb)
	u, _ = store.GetUser(groupID)
	if u.NotifyAtTime != true {
		t.Errorf("Expected group NotifyAtTime true after 2nd toggle, got %v", u.NotifyAtTime)
	}

	// 3. gtog_h_daily (было заблокировано в валидаторе префиксов callback!)
	cb = makeCallback(adminID, adminID, "gtog_h_daily:"+string("-100999"))
	b.handleCallback(cb)
	u, _ = store.GetUser(groupID)
	if u.HadithDailyEnabled != false {
		t.Errorf("Expected group HadithDailyEnabled false after toggle, got %v", u.HadithDailyEnabled)
	}
	b.handleCallback(cb)
	u, _ = store.GetUser(groupID)
	if u.HadithDailyEnabled != true {
		t.Errorf("Expected group HadithDailyEnabled true after 2nd toggle, got %v", u.HadithDailyEnabled)
	}

	// 4. gtog_h_pray (было заблокировано в валидаторе префиксов callback!)
	cb = makeCallback(adminID, adminID, "gtog_h_pray:"+string("-100999"))
	b.handleCallback(cb)
	u, _ = store.GetUser(groupID)
	if u.HadithPrayerEnabled != false {
		t.Errorf("Expected group HadithPrayerEnabled false after toggle, got %v", u.HadithPrayerEnabled)
	}
	b.handleCallback(cb)
	u, _ = store.GetUser(groupID)
	if u.HadithPrayerEnabled != true {
		t.Errorf("Expected group HadithPrayerEnabled true after 2nd toggle, got %v", u.HadithPrayerEnabled)
	}

	// 5. gtogsub
	cb = makeCallback(adminID, adminID, "gtogsub:"+string("-100999"))
	// изначально подписка активна (создана с дефолтным broadcast_times)
	_ = store.Subscribe(groupID, "Уфа")
	b.handleCallback(cb)
	u, _ = store.GetUser(groupID)
	if len(u.GetParsedBroadcastTimes()) != 0 {
		t.Errorf("Expected group broadcast times empty after unsubscribe, got %v", u.GetParsedBroadcastTimes())
	}
	b.handleCallback(cb)
	u, _ = store.GetUser(groupID)
	if len(u.GetParsedBroadcastTimes()) == 0 {
		t.Errorf("Expected group broadcast times restored after subscribe, got %v", u.GetParsedBroadcastTimes())
	}
}

// TestAdminHadiths_Checkmarks проверяет переключение галочек в админке хадисов
func TestAdminHadiths_Checkmarks(t *testing.T) {
	dbPath := "test_admin_hadiths_checkmarks.db"
	defer os.Remove(dbPath)
	b, store := newTestBot(t, dbPath)
	defer store.Close()

	adminID := int64(1001)

	// 1. Сборник Бухари
	cb := makeCallback(adminID, adminID, "adm_hadith_tog_col:bukhari")
	b.handleCallback(cb)
	if store.IsCollectionActive(storage.CollectionBukhari) {
		t.Errorf("Expected Bukhari to be disabled after toggle")
	}
	b.handleCallback(cb)
	if !store.IsCollectionActive(storage.CollectionBukhari) {
		t.Errorf("Expected Bukhari to be enabled after 2nd toggle")
	}

	// 2. Рассылка хадис дня
	cb = makeCallback(adminID, adminID, "adm_hadith_tog:daily")
	b.handleCallback(cb)
	if store.GetSetting("hadith_daily_enabled", "1") != "0" {
		t.Errorf("Expected hadith_daily_enabled to be '0'")
	}
	b.handleCallback(cb)
	if store.GetSetting("hadith_daily_enabled", "1") != "1" {
		t.Errorf("Expected hadith_daily_enabled to be '1'")
	}

	// 3. Рассылка к намазам
	cb = makeCallback(adminID, adminID, "adm_hadith_tog:prayer")
	b.handleCallback(cb)
	if store.GetSetting("hadith_prayer_enabled", "1") != "0" {
		t.Errorf("Expected hadith_prayer_enabled to be '0'")
	}
	b.handleCallback(cb)
	if store.GetSetting("hadith_prayer_enabled", "1") != "1" {
		t.Errorf("Expected hadith_prayer_enabled to be '1'")
	}
}

// TestFavoriteCities_Checkmarks проверяет, что при выборе города в дайджесте галочка ставится и убирается
func TestFavoriteCities_Checkmarks(t *testing.T) {
	dbPath := "test_fav_checkmarks.db"
	defer os.Remove(dbPath)
	b, store := newTestBot(t, dbPath)
	defer store.Close()

	userID := int64(55555)
	_ = store.SetUserCity(userID, "Уфа")

	// Нажимаем на город ID 23 (Стерлитамак) -> галочка ставится (добавляется в дайджест)
	cb := makeCallback(userID, userID, "fav_add_city:c:1:23")
	b.handleCallback(cb)

	favs, err := store.GetFavoriteCities(userID)
	if err != nil {
		t.Fatalf("GetFavoriteCities failed: %v", err)
	}
	found := false
	for _, c := range favs {
		if c == "Стерлитамак" {
			found = true
		}
	}
	if !found {
		t.Errorf("Expected 'Стерлитамак' to be added to favorites, got %v", favs)
	}

	// Нажимаем на тот же город повторно -> галочка убирается (удаляется из дайджеста)
	b.handleCallback(cb)

	favs, _ = store.GetFavoriteCities(userID)
	for _, c := range favs {
		if c == "Стерлитамак" {
			t.Errorf("Expected 'Стерлитамак' to be removed from favorites after 2nd click, got %v", favs)
		}
	}
}
