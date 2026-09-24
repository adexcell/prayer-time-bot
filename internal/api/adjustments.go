package api

import (
	"fmt"
	"strings"
	"time"
)

type PrayerRule struct {
	OffsetMinutes int
	FixedTime     string // например "13:30"
}

type AdjustmentStorage interface {
	GetActiveAdjustments(city string, date time.Time) (map[string]PrayerRule, error)
}

// ApplyOffset добавляет или вычитает минуты из строки времени "HH:MM"
func ApplyOffset(timeStr string, offsetMinutes int) string {
	if offsetMinutes == 0 || timeStr == "" {
		return timeStr
	}

	parts := strings.Split(strings.TrimSpace(timeStr), ":")
	if len(parts) < 2 {
		return timeStr
	}

	var hour, min int
	_, errH := fmt.Sscanf(parts[0], "%d", &hour)
	_, errM := fmt.Sscanf(parts[1], "%d", &min)
	if errH != nil || errM != nil {
		return timeStr
	}

	totalMinutes := hour*60 + min + offsetMinutes

	// Нормализация в диапазоне 0..1439 (24 часа)
	totalMinutes = (totalMinutes%(24*60) + (24 * 60)) % (24 * 60)

	newHour := totalMinutes / 60
	newMin := totalMinutes % 60

	return fmt.Sprintf("%02d:%02d", newHour, newMin)
}

// ApplyPrayerAdjustments применяет карту правил (смещений или фиксированного времени) к объекту расписания
func ApplyPrayerAdjustments(item *DUMRBItem, adjustments map[string]PrayerRule) {
	if len(adjustments) == 0 {
		return
	}

	apply := func(prayerNames []string, currentVal string) string {
		for _, name := range prayerNames {
			if rule, ok := adjustments[name]; ok {
				if rule.FixedTime != "" {
					return rule.FixedTime
				}
				if rule.OffsetMinutes != 0 {
					return ApplyOffset(currentVal, rule.OffsetMinutes)
				}
			}
		}
		if allRule, ok := adjustments["Все молитвы"]; ok {
			if allRule.FixedTime != "" {
				return allRule.FixedTime
			}
			if allRule.OffsetMinutes != 0 {
				return ApplyOffset(currentVal, allRule.OffsetMinutes)
			}
		}
		if allRule, ok := adjustments["all"]; ok {
			if allRule.FixedTime != "" {
				return allRule.FixedTime
			}
			if allRule.OffsetMinutes != 0 {
				return ApplyOffset(currentVal, allRule.OffsetMinutes)
			}
		}
		return currentVal
	}

	fajrBefore := item.Fajr
	item.Fajr = apply([]string{"Фаджр", "fajr"}, item.Fajr)
	if item.Fajr != fajrBefore && item.SuhurDo != "" {
		// Сдвигаем конец сухура
		for _, name := range []string{"Фаджр", "fajr", "Все молитвы", "all"} {
			if rule, ok := adjustments[name]; ok {
				if rule.FixedTime != "" {
					item.SuhurDo = rule.FixedTime
				} else if rule.OffsetMinutes != 0 {
					item.SuhurDo = ApplyOffset(item.SuhurDo, rule.OffsetMinutes)
				}
				break
			}
		}
	}

	item.Sunrise = apply([]string{"Восход", "sunrise"}, item.Sunrise)
	item.Dhuhr = apply([]string{"Зухр", "dhuhr"}, item.Dhuhr)
	item.Asr = apply([]string{"Аср", "asr"}, item.Asr)
	item.Maghrib = apply([]string{"Магриб", "maghrib"}, item.Maghrib)
	item.Isha = apply([]string{"Иша", "isha"}, item.Isha)
}
