package quality

import (
	"strings"
	"unicode"
)

type languageRules struct {
	actionRoots   []string
	resultRoots   []string
	firstPerson   map[string]struct{}
	fillerPhrases []string
}

var languageCatalog = map[string]languageRules{
	"en": {
		actionRoots: []string{
			"achiev", "analy", "architect", "automat", "build", "built", "creat", "cut", "deliver", "design",
			"develop", "direct", "drive", "establish", "execut", "expand", "generat", "implement", "improv", "increas",
			"launch", "lead", "led", "manag", "migrat", "optimiz", "orchestrat", "produc", "reduc", "resolv",
			"sav", "ship", "simplif", "streamlin", "strengthen", "transform", "upgrad",
		},
		resultRoots: []string{
			"accelerat", "boost", "cut", "deliver", "enable", "generat", "grow", "improv", "increas", "launch",
			"prevent", "reduc", "resolv", "result", "sav", "streamlin",
		},
		firstPerson: setOf("i", "we", "my", "our", "mine", "ours"),
		fillerPhrases: []string{
			"responsible for", "worked on", "helped with", "team player", "hard worker", "results-driven", "go-getter",
		},
	},
	"ru": {
		actionRoots: []string{
			"автоматиз", "анализир", "внедр", "возглав", "выстро", "достав", "запуст", "координир", "мигрир",
			"оптимиз", "организ", "повыс", "постро", "разработ", "реализ", "руковод", "сниз", "созда",
			"сократ", "спроектир", "ускор", "улучш", "устран",
		},
		resultRoots: []string{
			"автоматиз", "внедр", "запуст", "повыс", "позвол", "результ", "сниз", "сократ", "сэконом", "увелич",
			"улучш", "уменьш", "ускор", "устран",
		},
		firstPerson: setOf("я", "мы", "мой", "моя", "моё", "мое", "мои", "наш", "наша", "наше", "наши"),
		fillerPhrases: []string{
			"ответственный за", "работал над", "работала над", "помогал с", "помогала с", "командный игрок", "нацелен на результат", "нацелена на результат", "стрессоустойчив",
		},
	},
}

func normalizeLanguage(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if index := strings.IndexAny(value, "-_"); index >= 0 {
		value = value[:index]
	}
	if _, ok := languageCatalog[value]; ok {
		return value
	}
	if value == "" {
		return "en"
	}
	return value
}

func setOf(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func words(value string) []string {
	return strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
}

func hasRoot(value string, roots []string) bool {
	for _, token := range words(value) {
		for _, root := range roots {
			if strings.HasPrefix(token, root) {
				return true
			}
		}
	}
	return false
}

func startsWithRoot(value string, roots []string) bool {
	tokens := words(value)
	if len(tokens) == 0 {
		return false
	}
	for _, root := range roots {
		if strings.HasPrefix(tokens[0], root) {
			return true
		}
	}
	return false
}
