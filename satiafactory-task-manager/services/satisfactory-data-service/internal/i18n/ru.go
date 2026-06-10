package i18n

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"unicode"
)

var names = map[string]string{}

func Load(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("i18n: no ru names at %s: %v", path, err)
		return
	}
	if err := json.Unmarshal(data, &names); err != nil {
		log.Printf("i18n: parse error: %v", err)
		return
	}
	log.Printf("i18n: loaded %d Russian names", len(names))
}

func NameRU(className string) string {
	if v, ok := names[className]; ok {
		return v
	}
	return ""
}

func HasCyrillic(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Cyrillic, r) {
			return true
		}
	}
	return false
}

// SearchTerms returns query plus English hints for common Russian material words.
func SearchTerms(query string) []string {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil
	}
	seen := map[string]struct{}{q: {}}
	terms := []string{q}

	add := func(t string) {
		t = strings.TrimSpace(t)
		if t == "" {
			return
		}
		if _, ok := seen[t]; ok {
			return
		}
		seen[t] = struct{}{}
		terms = append(terms, t)
	}

	lower := strings.ToLower(q)
	ruToEn := map[string][]string{
		"желез": {"iron"},
		"сталь": {"steel"},
		"мед":   {"copper"},
		"алюмин": {"aluminum", "aluminium"},
		"пластин": {"plate"},
		"слиток": {"ingot"},
		"стержн": {"rod"},
		"винт":  {"screw"},
		"кабел": {"cable"},
		"провод": {"wire"},
		"бетон": {"concrete"},
		"ротор": {"rotor"},
		"двигател": {"motor"},
		"каркас": {"frame"},
		"пластик": {"plastic"},
		"резин": {"rubber"},
		"уголь": {"coal"},
		"нефт":  {"oil"},
		"топлив": {"fuel"},
		"кварц": {"quartz"},
		"кремн": {"silica"},
		"катерий": {"caterium"},
		"балка": {"beam"},
		"труб":  {"pipe"},
		"компьютер": {"computer"},
		"плат":  {"board", "plate"},
		"печат": {"circuit"},
		"армир": {"reinforced"},
		"модульн": {"modular"},
		"биомасс": {"biomass"},
		"ткан":  {"fabric"},
		"батар": {"battery"},
		"ракет": {"rocket"},
		"турбо": {"turbo"},
	}

	for key, en := range ruToEn {
		if strings.Contains(lower, key) {
			for _, e := range en {
				add(e)
			}
		}
	}

	enToRu := map[string][]string{
		"iron":     {"желез"},
		"steel":    {"сталь"},
		"copper":   {"мед"},
		"aluminum": {"алюмин"},
		"aluminium": {"алюмин"},
		"plate":    {"пластин"},
		"ingot":    {"слиток"},
		"rod":      {"стержн"},
		"screw":    {"винт"},
		"cable":    {"кабел"},
		"wire":     {"провод"},
		"concrete": {"бетон"},
		"rotor":    {"ротор"},
		"motor":    {"двигател"},
		"frame":    {"каркас"},
		"plastic":  {"пластик"},
		"rubber":   {"резин"},
		"coal":     {"уголь"},
		"oil":      {"нефт"},
		"fuel":     {"топлив"},
		"quartz":   {"кварц"},
		"silica":   {"кремн"},
		"caterium": {"катерий"},
		"beam":     {"балка"},
		"pipe":     {"труб"},
		"computer": {"компьютер"},
		"board":    {"плат", "печат"},
		"circuit":  {"печат", "плат"},
		"sheet":    {"лист"},
		"reinforced": {"армир"},
		"modular":  {"модульн"},
		"biomass":  {"биомасс"},
		"fabric":   {"ткан"},
		"battery":  {"батар"},
		"rocket":   {"ракет"},
		"turbo":    {"турбо"},
	}

	for key, ru := range enToRu {
		if strings.Contains(lower, key) {
			for _, r := range ru {
				add(r)
			}
		}
	}

	// Match Russian names directly
	for class, ru := range names {
		if strings.Contains(strings.ToLower(ru), lower) {
			add(class)
			parts := strings.Fields(ru)
			if len(parts) > 0 {
				add(parts[0])
			}
		}
		_ = class
	}

	return terms
}

func AllNames() map[string]string {
	return names
}

func IsAlternateRecipe(className, displayName string) bool {
	if strings.Contains(className, "Alternate") {
		return true
	}
	if strings.HasPrefix(displayName, "Alternate:") || strings.HasPrefix(displayName, "Alternate ") {
		return true
	}
	return false
}
