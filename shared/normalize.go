package genrenorm

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var caser = cases.Title(language.English)

func ToMixedCase(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	parts := strings.Fields(s)
	for i, p := range parts {
		runes := []rune(p)
		if len(runes) > 0 && unicode.IsLetter(runes[0]) {
			parts[i] = caser.String(p)
		}
	}
	return strings.Join(parts, " ")
}

var slashExpansions = map[string][]string{
	"death/thrash":              {"Death/Thrash", "Death Metal", "Thrash Metal"},
	"thrash/death":              {"Thrash/Death", "Thrash Metal", "Death Metal"},
	"death/doom":                {"Death/Doom", "Death Metal", "Doom Metal"},
	"doom/death":                {"Doom/Death", "Doom Metal", "Death Metal"},
	"black/death":               {"Black/Death", "Black Metal", "Death Metal"},
	"death/black":               {"Death/Black", "Death Metal", "Black Metal"},
	"black/thrash":              {"Black/Thrash", "Black Metal", "Thrash Metal"},
	"thrash/black":              {"Thrash/Black", "Thrash Metal", "Black Metal"},
	"death/grind":               {"Deathgrind", "Death Metal", "Grindcore"},
	"grind/death":               {"Deathgrind", "Grindcore", "Death Metal"},
	"death/hardcore":            {"Deathcore", "Death Metal", "Hardcore"},
	"black/grind":               {"Blackgrind", "Black Metal", "Grindcore"},
	"progressive death/thrash":  {"Progressive Death/Thrash", "Progressive Metal", "Death Metal", "Thrash Metal"},
	"blackened death/thrash":    {"Blackened Death/Thrash", "Black Metal", "Death Metal", "Thrash Metal"},
}

var genreSimpleExpansions = map[string]string{
	"thrash": "Thrash Metal", "death": "Death Metal", "black": "Black Metal",
	"doom": "Doom Metal", "heavy": "Heavy Metal", "hardcore": "Hardcore",
	"punk": "Punk", "folk": "Folk", "progressive": "Progressive", "power": "Power Metal",
	"symphonic": "Symphonic", "sludge": "Sludge", "stoner": "Stoner", "speed": "Speed Metal",
	"gothic": "Gothic", "groove": "Groove Metal", "funk": "Funk", "alternative": "Alternative",
	"indie": "Indie", "industrial": "Industrial", "math": "Mathcore", "horror": "Horror",
	"grindcore": "Grindcore", "crust": "Crust", "crossover": "Crossover",
	"metalcore": "Metalcore", "deathcore": "Deathcore",
}

var compoundExpansions = map[string][]string{
	"crossover thrash":    {"Crossover", "Thrash Metal"},
	"thrash crossover":    {"Thrash Metal", "Crossover"},
	"beatdown hardcore":   {"Beatdown", "Hardcore"},
	"hardcore beatdown":   {"Hardcore", "Beatdown"},
	"sludge metal":        {"Sludge Metal"},
	"stoner metal":        {"Stoner Metal"},
	"stoner rock":         {"Stoner Rock"},
	"death metal":         {"Death Metal"},
	"thrash metal":        {"Thrash Metal"},
	"black metal":         {"Black Metal"},
	"doom metal":          {"Doom Metal"},
	"heavy metal":         {"Heavy Metal"},
	"hard rock":           {"Hard Rock"},
	"classic rock":        {"Classic Rock"},
	"punk rock":           {"Punk Rock"},
	"alternative rock":    {"Alternative Rock"},
	"progressive metal":   {"Progressive Metal"},
	"power metal":         {"Power Metal"},
	"folk metal":          {"Folk Metal"},
	"symphonic metal":     {"Symphonic Metal"},
	"gothic metal":        {"Gothic Metal"},
	"groove metal":        {"Groove Metal"},
	"speed metal":         {"Speed Metal"},
	"funk metal":          {"Funk Metal"},
	"indie rock":          {"Indie Rock"},
	"hardcore punk":       {"Hardcore Punk"},
	"death metal/grindcore":   {"Deathgrind", "Death Metal", "Grindcore"},
	"grindcore/death metal":   {"Deathgrind", "Grindcore", "Death Metal"},
	"black/death metal":       {"Black/Death", "Black Metal", "Death Metal"},
	"death/thrash metal":      {"Death/Thrash", "Death Metal", "Thrash Metal"},
	"thrash/death metal":      {"Death/Thrash", "Thrash Metal", "Death Metal"},
	"new wave":             {"New Wave"},
	"indie pop":            {"Indie Pop"},
	"psychedelic pop":      {"Psychedelic Pop"},
	"synth pop":            {"Synth-Pop"},
	"synthpop":             {"Synth-Pop"},
	"power violence":       {"Powerviolence"},
	"raw black metal":      {"Black Metal"},
	"epic doom metal":      {"Doom Metal"},
	"psychedelic trance":   {"Psytrance"},
	"drum and bass":        {"Drum & Bass"},
	"drum n bass":          {"Drum & Bass"},
	"drum & bass":          {"Drum & Bass"},
}

var fixGenreMappings = map[string]string{
	"rnb":       "R&B",
	"r&b":       "R&B",
	"hiphop":    "Hip-Hop",
	"hip hop":   "Hip-Hop",
	"synthpop":  "Synth-Pop",
	"synth pop": "Synth-Pop",
	"kpop":      "K-Pop",
	"k-pop":     "K-Pop",
	"avant-garde": "Avantgarde",
	"avantgarde":  "Avantgarde",
	"psy-trance":  "Psytrance",
}

func FixGenreName(genre string) string {
	lower := strings.ToLower(strings.TrimSpace(genre))
	if v, ok := fixGenreMappings[lower]; ok {
		return v
	}
	return ToMixedCase(genre)
}

var qualifierRegex = regexp.MustCompile(`\s*\([^)]*\)\s*`)

func StripQualifiers(genre string) string {
	return strings.TrimSpace(qualifierRegex.ReplaceAllString(genre, " "))
}

func UniqueCaseInsensitive(arr []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, item := range arr {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key := strings.ToLower(strings.NewReplacer("-", " ").Replace(item))
		key = strings.Join(strings.Fields(key), " ")
		if !seen[key] {
			seen[key] = true
			out = append(out, item)
		}
	}
	return out
}

func ExpandGenre(genre string) []string {
	lower := strings.ToLower(strings.TrimSpace(genre))
	if v, ok := slashExpansions[lower]; ok {
		return v
	}
	if v, ok := compoundExpansions[lower]; ok {
		return v
	}

	slashIdx := strings.IndexByte(lower, '/')
	if slashIdx >= 0 {
		p1 := strings.TrimSpace(lower[:slashIdx])
		p2 := strings.TrimSpace(lower[slashIdx+1:])
		expanded := append(ExpandGenre(p1), ExpandGenre(p2)...)
		return UniqueCaseInsensitive(expanded)
	}

	if v, ok := genreSimpleExpansions[lower]; ok {
		return []string{v}
	}
	return []string{FixGenreName(genre)}
}

func RemoveRedundantGenres(genres []string) []string {
	if len(genres) <= 1 {
		return genres
	}
	lowerGenres := make([]string, len(genres))
	for i, g := range genres {
		lowerGenres[i] = strings.ToLower(g)
	}
	var filtered []string
	for i, current := range genres {
		currentLower := lowerGenres[i]
		if currentLower == "southern" {
			continue
		}
		if currentLower == "nu" {
			filtered = append(filtered, "Nu Metal")
			continue
		}
		if strings.Contains(current, "/") {
			filtered = append(filtered, current)
			continue
		}
		isRedundant := false
		for j, other := range genres {
			if i == j {
				continue
			}
			otherLower := lowerGenres[j]
			if strings.Contains(other, "/") {
				continue
			}
			if len(other) > len(current) {
				escaped := regexp.QuoteMeta(currentLower)
				var re *regexp.Regexp
				if strings.Contains(currentLower, " ") {
					re = regexp.MustCompile(`(?i)(^|\s)` + escaped + `(\s|$)`)
				} else {
					re = regexp.MustCompile(`(?i)\b` + escaped + `\b`)
				}
				if re.MatchString(otherLower) {
					isRedundant = true
					break
				}
			}
		}
		if !isRedundant {
			filtered = append(filtered, current)
		}
	}
	return filtered
}

func CleanupParentGenres(genres []string) []string {
	lower := make([]string, len(genres))
	for i, g := range genres {
		lower[i] = strings.ToLower(g)
	}

	hasMetalSub := false
	for _, g := range lower {
		if g != "metal" && g != "heavy metal" && strings.Contains(g, "metal") {
			hasMetalSub = true
			break
		}
	}
	hasIndMetal := false
	for _, g := range lower {
		if g == "industrial metal" {
			hasIndMetal = true
			break
		}
	}
	var out []string
	for i, g := range genres {
		lo := lower[i]
		if lo == "metal" && hasMetalSub {
			continue
		}
		if lo == "industrial" && hasIndMetal {
			continue
		}
		if lo == "old school death metal" {
			out = append(out, "Death Metal")
			continue
		}
		out = append(out, g)
	}
	hasGrind := false
	for _, g := range lower {
		if g == "grindcore" {
			hasGrind = true
			break
		}
	}
	hasDeath := false
	for _, g := range lower {
		if g == "death metal" || g == "deathcore" {
			hasDeath = true
			break
		}
	}

	hasGoregrind := false
	for _, g := range lower {
		if g == "goregrind" {
			hasGoregrind = true
			break
		}
	}

	if hasGoregrind {
		var filtered []string
		for _, g := range out {
			if strings.ToLower(g) != "deathgrind" && strings.ToLower(g) != "gore/grind" {
				filtered = append(filtered, g)
			}
		}
		out = filtered
	}

	if hasGrind && hasDeath && !hasGoregrind {
		hasDeathgrind := false
		for _, g := range lower {
			if g == "deathgrind" || strings.Contains(g, "deathgrind") {
				hasDeathgrind = true
				break
			}
		}
		if !hasDeathgrind {
			out = append([]string{"Deathgrind"}, out...)
		}
	}
	return out
}

func AddParentMetal(genres []string) []string {
	lower := make([]string, len(genres))
	for i, g := range genres {
		lower[i] = strings.ToLower(g)
	}
	var out []string
	added := make(map[string]bool)
	for i, g := range genres {
		lo := lower[i]
		replaced := false
		for base, parent := range map[string]string{
			"death metal":  "Death Metal",
			"black metal":  "Black Metal",
			"thrash metal": "Thrash Metal",
			"doom metal":   "Doom Metal",
		} {
			if lo != base && strings.HasSuffix(lo, " "+base) {
				if !added[parent] {
					out = append(out, parent)
					added[parent] = true
				}
				replaced = true
				break
			}
		}
		if !replaced {
			out = append(out, g)
		}
	}
	return out
}

func ExpandGenres(genres []string) []string {
	var all []string
	for _, g := range genres {
		all = append(all, ExpandGenre(g)...)
	}
	all = UniqueCaseInsensitive(all)
	all = AddParentMetal(all)

	lower := make([]string, len(all))
	for i, g := range all {
		lower[i] = strings.ToLower(g)
	}

	hasDeath := false
	hasDoom := false
	hasThrash := false
	hasBlack := false
	for _, g := range lower {
		switch g {
		case "death metal":
			hasDeath = true
		case "doom metal":
			hasDoom = true
		case "thrash metal":
			hasThrash = true
		case "black metal":
			hasBlack = true
		}
	}

	deathDoom := false
	deathThrash := false
	blackThrash := false
	for _, g := range lower {
		if strings.Contains(g, "death/doom") || strings.Contains(g, "doom/death") {
			deathDoom = true
		}
		if strings.Contains(g, "death/thrash") || strings.Contains(g, "thrash/death") {
			deathThrash = true
		}
		if strings.Contains(g, "black/thrash") || strings.Contains(g, "thrash/black") {
			blackThrash = true
		}
	}

	if hasDeath && hasDoom && !deathDoom {
		all = append([]string{"Death/Doom"}, all...)
	}
	if hasThrash && hasDeath && !deathThrash {
		all = append([]string{"Death/Thrash"}, all...)
	}
	if hasBlack && hasThrash && !blackThrash {
		all = append([]string{"Black/Thrash"}, all...)
	}

	result := UniqueCaseInsensitive(all)
	result = RemoveRedundantGenres(result)
	result = CleanupParentGenres(result)
	result = UniqueCaseInsensitive(result)

	for i, g := range result {
		lo := strings.ToLower(strings.NewReplacer("-", " ").Replace(g))
		switch {
		case lo == "hip hop" || lo == "hiphop" || lo == "uk hip hop" || lo == "underground hip hop":
			result[i] = "Hip-Hop"
		case lo == "rnb" || lo == "r&b":
			result[i] = "R&B"
		case lo == "kpop" || lo == "k pop":
			result[i] = "K-Pop"
		case lo == "synth pop" || lo == "synthpop":
			result[i] = "Synth-Pop"
		}
	}

	result = combineAdjacentGenres(result)
	result = UniqueCaseInsensitive(result)
	return result
}

func combineAdjacentGenres(genres []string) []string {
	// Rap → Hip-Hop (always)
	for i, g := range genres {
		if strings.ToLower(g) == "rap" {
			genres[i] = "Hip-Hop"
		}
	}

	// Remove Traditional prefix
	for i, g := range genres {
		lo := strings.ToLower(g)
		if strings.HasPrefix(lo, "traditional ") {
			genres[i] = strings.TrimSpace(g[len("Traditional "):])
		}
	}

	hasHH := false
	for _, g := range genres {
		if strings.ToLower(g) == "hip-hop" {
			hasHH = true
			break
		}
	}

	lower := make([]string, len(genres))
	for i, g := range genres {
		lower[i] = strings.ToLower(g)
	}

	type pair struct {
		adjective string
		genre     string
		result    string
	}
	pairs := []pair{
		{"technical", "death metal", "Technical Death Metal"},
		{"melodic", "thrash metal", "Melodic Thrash Metal"},
		{"progressive", "metal", "Progressive Metal"},
		{"slam", "death metal", "Slamming Brutal Death Metal"},
	}

	consumed := make(map[int]bool)
	var additions []string

	for _, p := range pairs {
		adjIdx, genreIdx := -1, -1
		for i, g := range lower {
			if consumed[i] {
				continue
			}
			if g == p.adjective {
				adjIdx = i
			}
			if g == p.genre {
				genreIdx = i
			}
		}
		if adjIdx >= 0 && genreIdx >= 0 {
			consumed[adjIdx] = true
			consumed[genreIdx] = true
			additions = append(additions, p.result)
		}
	}

	var out []string
	for i, g := range genres {
		if consumed[i] {
			continue
		}
		if hasHH && lower[i] == "beat" {
			continue
		}
		out = append(out, g)
	}
	out = append(out, additions...)

	// Add Hip-Hop if Horrorcore present and not already there
	hasHorrorcore := false
	hasHH = false
	for _, g := range out {
		lo := strings.ToLower(g)
		if lo == "horrorcore" {
			hasHorrorcore = true
		}
		if lo == "hip-hop" {
			hasHH = true
		}
	}
	if hasHorrorcore && !hasHH {
		for i, g := range out {
			if strings.ToLower(g) == "horrorcore" {
				out = append(out[:i], append([]string{"Hip-Hop"}, out[i:]...)...)
				break
			}
		}
	}

	return UniqueCaseInsensitive(out)
}

func NormalizeGenres(rawGenre string, maxGenres int) []string {
	cleaned := StripQualifiers(rawGenre)
	var parts []string
	for _, p := range strings.FieldsFunc(cleaned, func(r rune) bool {
		return r == ';' || r == ','
	}) {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	expanded := ExpandGenres(parts)
	if len(expanded) > maxGenres {
		expanded = expanded[:maxGenres]
	}
	return expanded
}
