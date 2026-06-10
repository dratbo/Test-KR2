package icons

import (
	"regexp"
	"strings"
	"unicode"
)

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

const BaseURL = "https://raw.githubusercontent.com/greeny/SatisfactoryTools/u4-dev/www/assets/images/items/"

func SlugFromDisplayName(name string) string {
	if name == "" {
		return ""
	}
	s := strings.ToLower(strings.TrimSpace(name))
	s = nonAlphaNum.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func SlugFromClassName(className string) string {
	s := strings.TrimSuffix(className, "_C")
	s = strings.TrimPrefix(s, "Desc_")
	return camelToKebab(s)
}

func camelToKebab(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		if r == '_' {
			b.WriteByte('-')
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

func URLsForItem(displayName, className string) []string {
	seen := make(map[string]struct{})
	var urls []string
	add := func(slug string) {
		if slug == "" {
			return
		}
		if _, ok := seen[slug]; ok {
			return
		}
		seen[slug] = struct{}{}
		urls = append(urls, BaseURL+slug+"_64.png")
	}
	add(SlugFromDisplayName(displayName))
	add(SlugFromClassName(className))
	return urls
}
