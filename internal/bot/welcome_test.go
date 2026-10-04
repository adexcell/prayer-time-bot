package bot

import (
	"os"
	"strings"
	"testing"

	"namaz-time-bot/internal/storage"
)

func TestBot_GetWelcomeMessage(t *testing.T) {
	dbPath := "test_welcome_bot.db"
	defer os.Remove(dbPath)

	store, err := storage.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to init store: %v", err)
	}
	defer store.Close()

	b := &Bot{
		storage: store,
	}

	// 1. Default welcome message
	msg := b.getWelcomeMessage("Уфа")
	if !strings.Contains(msg, "📍 Ваш текущий выбор: *Уфа*") {
		t.Errorf("Expected default message to contain current city, got: %s", msg)
	}

	// 2. Custom welcome message with {city}
	_ = store.SetWelcomeMessage("Салям! Добро пожаловать. Город: {city}")
	msg = b.getWelcomeMessage("Стерлитамак")
	expected := "Салям! Добро пожаловать. Город: Стерлитамак"
	if msg != expected {
		t.Errorf("Expected %q, got %q", expected, msg)
	}

	// 3. Custom welcome message with {город}
	_ = store.SetWelcomeMessage("Добро пожаловать в {город}!")
	msg = b.getWelcomeMessage("Нефтекамск")
	expected = "Добро пожаловать в Нефтекамск!"
	if msg != expected {
		t.Errorf("Expected %q, got %q", expected, msg)
	}

	// 4. Custom welcome message without tags (appends city info)
	_ = store.SetWelcomeMessage("Приветственный текст без тегов")
	msg = b.getWelcomeMessage("Салават")
	if !strings.HasPrefix(msg, "Приветственный текст без тегов") {
		t.Errorf("Expected prefix, got: %s", msg)
	}
	if !strings.Contains(msg, "📍 Ваш текущий выбор: *Салават*") {
		t.Errorf("Expected appended city block, got: %s", msg)
	}

	// 5. Reset to default
	_ = store.SetWelcomeMessage("")
	msg = b.getWelcomeMessage("Белорецк")
	if !strings.Contains(msg, "📍 Ваш текущий выбор: *Белорецк*") {
		t.Errorf("Expected default message after reset, got: %s", msg)
	}
}
