package presence

import (
	"path"
	"strings"
)

var mapNames = map[string]string{
	"de_ancient":  "Ancient",
	"de_anubis":   "Anubis",
	"de_dust2":    "Dust II",
	"de_inferno":  "Inferno",
	"de_mirage":   "Mirage",
	"de_nuke":     "Nuke",
	"de_overpass": "Overpass",
	"de_train":    "Train",
	"de_vertigo":  "Vertigo",
	"de_basalt":   "Basalt",
	"de_edin":     "Edin",
	"de_palais":   "Palais",
	"de_whistle":  "Whistle",
	"de_thera":    "Thera",
	"de_mills":    "Mills",
	"de_assembly": "Assembly",
	"de_memento":  "Memento",
	"de_cache":    "Cache",
	"de_grail":    "Grail",
	"de_jura":     "Jura",
	"de_brewery":  "Brewery",
	"de_dogtown":  "Dogtown",
	"cs_italy":    "Italy",
	"cs_office":   "Office",
	"cs_agency":   "Agency",
	"ar_baggage":  "Baggage",
	"ar_shoots":   "Shoots",
	"ar_pool_day": "Pool Day",
}

var modeNames = map[string]string{
	"competitive":        "Competitive",
	"premier":            "Premier",
	"casual":             "Casual",
	"deathmatch":         "Deathmatch",
	"scrimcomp2v2":       "Wingman",
	"gungameprogressive": "Arms Race",
	"training":           "Training",
	"custom":             "Custom",
}

var mapPrefixes = []string{"de_", "cs_", "ar_", "dz_", "gd_"}

func mapKey(name string) string {
	return strings.ToLower(path.Base(name))
}

func mapTitle(key string) string {
	if title, ok := mapNames[key]; ok {
		return title
	}

	for _, prefix := range mapPrefixes {
		key = strings.TrimPrefix(key, prefix)
	}

	return titleCase(key)
}

func modeTitle(mode string) string {
	if title, ok := modeNames[mode]; ok {
		return title
	}

	return titleCase(mode)
}

func titleCase(s string) string {
	words := strings.Fields(strings.ReplaceAll(s, "_", " "))

	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}

	return strings.Join(words, " ")
}
