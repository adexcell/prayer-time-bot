package api

import (
	"fmt"
	"strings"
	"time"
)

type AdjustmentStorage interface {
	GetActiveAdjustments(city string, date time.Time) (map[string]int, error)
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

// ApplyPrayerAdjustments применяет карту смещений к объекту расписания
func ApplyPrayerAdjustments(item *DUMRBItem, adjustments map[string]int) {
	if len(adjustments) == 0 {
		return
	}

	apply := func(prayerNames []string, currentVal string) string {
		for _, name := range prayerNames {
			if offset, ok := adjustments[name]; ok && offset != 0 {
				return ApplyOffset(currentVal, offset)
			}
		}
		if allOffset, ok := adjustments["Все молитвы"]; ok && allOffset != 0 {
			return ApplyOffset(currentVal, allOffset)
		}
		if allOffset, ok := adjustments["all"]; ok && allOffset != 0 {
			return ApplyOffset(currentVal, allOffset)
		}
		return currentVal
	}

	fajrBefore := item.Fajr
	item.Fajr = apply([]string{"Фаджр", "fajr"}, item.Fajr)
	if item.Fajr != fajrBefore && item.SuhurDo != "" {
		// Сдвигаем конец сухура на то же смещение, что и Фаджр
		for _, name := range []string{"Фаджр", "fajr", "Все молитвы", "all"} {
			if offset, ok := adjustments[name]; ok && offset != 0 {
				item.SuhurDo = ApplyOffset(item.SuhurDo, offset)
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
