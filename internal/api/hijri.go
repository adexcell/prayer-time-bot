package api

import (
	"fmt"
	"sync"
	"time"

	"github.com/hablullah/go-hijri"
)

var (
	globalHijriOffsetMutex sync.RWMutex
	globalHijriOffset      int
)

// SetGlobalHijriOffset устанавливает глобальное смещение мусульманского календаря в днях
func SetGlobalHijriOffset(offset int) {
	globalHijriOffsetMutex.Lock()
	defer globalHijriOffsetMutex.Unlock()
	globalHijriOffset = offset
}

// GetGlobalHijriOffset возвращает текущее глобальное смещение мусульманского календаря в днях
func GetGlobalHijriOffset() int {
	globalHijriOffsetMutex.RLock()
	defer globalHijriOffsetMutex.RUnlock()
	return globalHijriOffset
}

// HijriDate содержит число, месяц, год по мусульманскому календарю
type HijriDate struct {
	Day     int
	Month   int
	Year    int
	Weekday time.Weekday
}

var hijriMonthNamesRu = [13]string{
	"",
	"Мухаррам",
	"Сафар",
	"Раби аль-авваль",
	"Раби ас-сани",
	"Джумада аль-уля",
	"Джумада аль-ахира",
	"Раджаб",
	"Шаабан",
	"Рамадан",
	"Шавваль",
	"Зуль-каада",
	"Зуль-хиджа",
}

var hijriMonthNamesBa = [13]string{
	"",
	"Мөхәррәм",
	"Сәфәр",
	"Рабиғел-әүвәл",
	"Рабиғел-ахыр",
	"Йомадиел-әүвәл",
	"Йомадиел-ахыр",
	"Рәжәп",
	"Шәғбан",
	"Рамаҙан",
	"Шәүвәл",
	"Зөлҡәғиҙә",
	"Зөлхизә",
}

var hijriMonthNamesAr = [13]string{
	"",
	"محرم",
	"صفر",
	"ربيع الأول",
	"ربيع الثاني",
	"جمادى الأولى",
	"جمادى الآخرة",
	"رجب",
	"شعبان",
	"رمضان",
	"شوال",
	"ذو القعدة",
	"ذو الحجة",
}

// GetHijriMonthName возвращает название месяца по Хиджре для выбранного пресета языка
func GetHijriMonthName(month int, preset string) string {
	if month < 1 || month > 12 {
		return ""
	}
	switch preset {
	case "ba":
		return hijriMonthNamesBa[month]
	case "ar":
		return hijriMonthNamesAr[month]
	case "ru_ar":
		return fmt.Sprintf("%s (%s)", hijriMonthNamesRu[month], hijriMonthNamesAr[month])
	default: // "ru"
		return hijriMonthNamesRu[month]
	}
}

// GetHijriDate возвращает дату по Хиджре с учетом переданного смещения в днях
func GetHijriDate(t time.Time, offsetDays int) (HijriDate, error) {
	shifted := t.AddDate(0, 0, offsetDays)
	uq, err := hijri.CreateUmmAlQuraDate(shifted)
	if err != nil {
		return HijriDate{}, err
	}
	return HijriDate{
		Day:     int(uq.Day),
		Month:   int(uq.Month),
		Year:    int(uq.Year),
		Weekday: uq.Weekday,
	}, nil
}

// FindOffsetForTargetDay ищет смещение в днях (-5..+5), при котором дата t дает targetDay число месяца
func FindOffsetForTargetDay(t time.Time, targetDay int) (int, error) {
	if targetDay < 1 || targetDay > 30 {
		return 0, fmt.Errorf("число месяца должно быть от 1 до 30")
	}

	// Ищем смещение в порядке минимального отклонения от сегодняшней даты: 0, +1, -1, +2, -2...
	for _, offset := range []int{0, 1, -1, 2, -2, 3, -3, 4, -4, 5, -5} {
		hd, err := GetHijriDate(t, offset)
		if err == nil && hd.Day == targetDay {
			return offset, nil
		}
	}
	return 0, fmt.Errorf("не удалось найти подходящее смещение для дня %d", targetDay)
}

// FormatHijriDate возвращает отформатированную строку даты по Хиджре (например, "18 Раби ас-сани 1448 г. х.")
func FormatHijriDate(t time.Time, preset string, explicitOffset ...int) string {
	offset := GetGlobalHijriOffset()
	if len(explicitOffset) > 0 && explicitOffset[0] != 0 {
		offset = explicitOffset[0]
	}

	hd, err := GetHijriDate(t, offset)
	if err != nil {
		return ""
	}

	month := hd.Month
	if month < 1 || month > 12 {
		return ""
	}

	switch preset {
	case "ar":
		return fmt.Sprintf("%d %s %d هـ", hd.Day, hijriMonthNamesAr[month], hd.Year)
	case "ba":
		return fmt.Sprintf("%d %s %d һ. й.", hd.Day, hijriMonthNamesBa[month], hd.Year)
	case "ru_ar":
		return fmt.Sprintf("%d %s (%s) %d г. х.", hd.Day, hijriMonthNamesRu[month], hijriMonthNamesAr[month], hd.Year)
	default: // "ru"
		return fmt.Sprintf("%d %s %d г. х.", hd.Day, hijriMonthNamesRu[month], hd.Year)
	}
}
