package storage

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

//go:embed asmaul_husna.json
var embeddedAsmaJSON []byte

type AsmaImage struct {
	SVGCardPath         string  `json:"svg_card_path"`
	CalligraphySVGURL   *string `json:"calligraphy_svg_url"`
	CalligraphyPNGURL   *string `json:"calligraphy_png_url"`
	LocalCalligraphyPNG string  `json:"local_calligraphy_png,omitempty"`
}

type AsmaRelatedName struct {
	ID              int    `json:"id"`
	Arabic          string `json:"arabic"`
	Transliteration string `json:"transliteration"`
	Translation     string `json:"translation"`
}

type AsmaQuranQuote struct {
	Surah int    `json:"surah"`
	Ayah  int    `json:"ayah"`
	Text  string `json:"text"`
}

type AsmaQuran struct {
	MentionsCount    int              `json:"mentions_count"`
	References       []string         `json:"references"`
	InlineReferences []string         `json:"inline_references"`
	Quotes           []AsmaQuranQuote `json:"quotes"`
}

type AsmaName struct {
	ID                int               `json:"id"`
	Name              string            `json:"name"`
	Arabic            string            `json:"arabic"`
	Translation       string            `json:"translation"`
	TransliterationEn string            `json:"transliteration_en"`
	EnglishMeaning    string            `json:"english_meaning"`
	Image             AsmaImage         `json:"image"`
	Meaning           string            `json:"meaning"`
	RelatedNames      []AsmaRelatedName `json:"related_names"`
	Quran             AsmaQuran         `json:"quran"`
	HadithReferences  []string          `json:"hadith_references"`
	SourceURL         string            `json:"source_url"`
}

var (
	asmaList []*AsmaName
	asmaMap  map[int]*AsmaName
	asmaOnce sync.Once
)

func initAsma() {
	asmaOnce.Do(func() {
		var rawData []byte
		// Сначала пробуем прочитать свежий файл из папки data
		if data, err := os.ReadFile("data/asmaul_husna/asmaul_husna.json"); err == nil && len(data) > 0 {
			rawData = data
		} else {
			rawData = embeddedAsmaJSON
		}

		if len(rawData) == 0 {
			return
		}

		var list []*AsmaName
		if err := json.Unmarshal(rawData, &list); err == nil {
			asmaList = list
			asmaMap = make(map[int]*AsmaName, len(list))
			for _, item := range list {
				asmaMap[item.ID] = item
			}
		}
	})
}

// GetAllAsmaNames возвращает список всех имён Аллаха
func GetAllAsmaNames() []*AsmaName {
	initAsma()
	return asmaList
}

// GetAsmaName возвращает имя Аллаха по его ID (1..100)
func GetAsmaName(id int) (*AsmaName, error) {
	initAsma()
	if item, exists := asmaMap[id]; exists {
		return item, nil
	}
	return nil, fmt.Errorf("имя с ID %d не найдено", id)
}
