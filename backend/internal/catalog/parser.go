package catalog

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Kind string

const (
	KindUnknown Kind = "unknown"
	KindMovie   Kind = "movie"
	KindSeries  Kind = "series"
	KindAnime   Kind = "anime"
)

type ParsedMedia struct {
	Kind            Kind   `json:"kind"`
	Title           string `json:"title"`
	Year            int    `json:"year,omitempty"`
	Season          int    `json:"season,omitempty"`
	Episode         int    `json:"episode,omitempty"`
	AbsoluteEpisode int    `json:"absoluteEpisode,omitempty"`
	Quality         string `json:"quality,omitempty"`
	CanonicalKey    string `json:"canonicalKey"`
}

var (
	seriesPattern  = regexp.MustCompile(`(?i)(.*?)[ ._\-]+S(\d{1,2})E(\d{1,3})`)
	yearPattern    = regexp.MustCompile(`\b(19\d{2}|20\d{2})\b`)
	qualityPattern = regexp.MustCompile(
		`(?i)\b(2160p|1080p|720p|480p|uhd|hdr|dv|bluray|web-dl|webrip|hdtv|x265|x264|hevc|av1)\b`,
	)
	animePattern = regexp.MustCompile(`(?i)^(?:\[[^\]]+\][ ._-]*)?(.+?)[ ._-]+-[ ._-]*(\d{1,4})(?:\D|$)`)
)

func ParseMediaPath(path string) ParsedMedia {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	quality := detectQuality(name)
	lowerPath := strings.ToLower(filepath.ToSlash(path))

	if match := seriesPattern.FindStringSubmatch(name); len(match) == 4 {
		season, _ := strconv.Atoi(match[2])
		episode, _ := strconv.Atoi(match[3])
		title := cleanTitle(match[1])
		return ParsedMedia{
			Kind:         KindSeries,
			Title:        title,
			Season:       season,
			Episode:      episode,
			Quality:      quality,
			CanonicalKey: "series:" + slug(title) + ":s" + pad2(season) + "e" + pad2(episode),
		}
	}

	if strings.Contains(lowerPath, "/anime/") {
		if match := animePattern.FindStringSubmatch(name); len(match) == 3 {
			episode, _ := strconv.Atoi(match[2])
			title := cleanTitle(match[1])
			return ParsedMedia{
				Kind:            KindAnime,
				Title:           title,
				AbsoluteEpisode: episode,
				Quality:         quality,
				CanonicalKey:    "anime:" + slug(title) + ":e" + pad3(episode),
			}
		}
	}

	if match := yearPattern.FindStringSubmatchIndex(name); len(match) >= 4 {
		year, _ := strconv.Atoi(name[match[2]:match[3]])
		title := cleanTitle(name[:match[0]])
		return ParsedMedia{
			Kind:         KindMovie,
			Title:        title,
			Year:         year,
			Quality:      quality,
			CanonicalKey: "movie:" + slug(title) + ":" + strconv.Itoa(year),
		}
	}

	title := cleanTitle(name)
	return ParsedMedia{
		Kind:         KindUnknown,
		Title:        title,
		Quality:      quality,
		CanonicalKey: "unknown:" + slug(title),
	}
}

func detectQuality(value string) string {
	if match := qualityPattern.FindString(value); match != "" {
		return strings.ToLower(match)
	}
	return ""
}

func cleanTitle(value string) string {
	value = regexp.MustCompile(`^\[[^\]]+\][ ._-]*`).ReplaceAllString(value, "")
	value = regexp.MustCompile(`\([^)]+\)`).ReplaceAllString(value, " ")
	value = strings.NewReplacer(".", " ", "_", " ", "-", " ").Replace(value)
	value = qualityPattern.ReplaceAllString(value, " ")
	value = strings.Join(strings.Fields(value), " ")
	return strings.TrimSpace(value)
}

func slug(value string) string {
	value = strings.ToLower(value)
	var out strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			out.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(out.String(), "-")
}

func pad2(value int) string {
	if value < 10 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}

func pad3(value int) string {
	if value < 10 {
		return "00" + strconv.Itoa(value)
	}
	if value < 100 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}
