package scraper

import (
	"fmt"
	"regexp"
	"strings"
)

const AVProvider = "av"
const AVSourceAuto = "auto"

type AVNumber struct {
	Raw                string   `json:"raw"`
	Normalized         string   `json:"normalized"`
	Kind               string   `json:"kind"`
	Prefix             string   `json:"prefix"`
	Digits             string   `json:"digits"`
	PreferredProviders []string `json:"preferred_providers"`
}

var (
	avFC2Pattern     = regexp.MustCompile(`(?i)\bFC2(?:[-_\s]?PPV)?[-_\s]?(\d{5,})\b`)
	avHeyzoPattern   = regexp.MustCompile(`(?i)\bHEYZO[-_\s]?(\d{3,})\b`)
	avCaribPattern   = regexp.MustCompile(`(?i)\b(?:CARIB|1PONDO|10MUSUME)[-_\s]?(\d{6})[-_\s]?(\d{2,3})\b`)
	avStandardPatten = regexp.MustCompile(`(?i)\b([A-Z]{2,8})[-_\s]?(\d{2,6})\b`)
)

func ParseAVNumber(value string) (AVNumber, bool) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return AVNumber{}, false
	}
	if match := avFC2Pattern.FindStringSubmatch(raw); len(match) == 2 {
		digits := match[1]
		return AVNumber{
			Raw:                raw,
			Normalized:         "FC2-PPV-" + digits,
			Kind:               "fc2",
			Prefix:             "FC2-PPV",
			Digits:             digits,
			PreferredProviders: []string{"fc2", "javdb", "javbus"},
		}, true
	}
	if match := avHeyzoPattern.FindStringSubmatch(raw); len(match) == 2 {
		digits := match[1]
		return AVNumber{
			Raw:                raw,
			Normalized:         "HEYZO-" + digits,
			Kind:               "heyzo",
			Prefix:             "HEYZO",
			Digits:             digits,
			PreferredProviders: []string{"heyzo", "javdb", "javbus"},
		}, true
	}
	if match := avCaribPattern.FindStringSubmatch(raw); len(match) == 3 {
		prefix := strings.ToUpper(strings.FieldsFunc(match[0], func(r rune) bool {
			return r == '-' || r == '_' || r == ' '
		})[0])
		normalized := fmt.Sprintf("%s-%s-%s", prefix, match[1], match[2])
		return AVNumber{
			Raw:                raw,
			Normalized:         normalized,
			Kind:               strings.ToLower(prefix),
			Prefix:             prefix,
			Digits:             match[1] + "-" + match[2],
			PreferredProviders: []string{strings.ToLower(prefix), "javdb"},
		}, true
	}
	if match := avStandardPatten.FindStringSubmatch(raw); len(match) == 3 {
		prefix := strings.ToUpper(match[1])
		digits := strings.TrimLeft(match[2], "0")
		if digits == "" {
			digits = "0"
		}
		return AVNumber{
			Raw:                raw,
			Normalized:         prefix + "-" + match[2],
			Kind:               "standard",
			Prefix:             prefix,
			Digits:             digits,
			PreferredProviders: []string{"javdb", "javbus", "r18", "jav321"},
		}, true
	}
	return AVNumber{Raw: raw}, false
}

func SelectAVLiveSource(parsed AVNumber, requested string) (string, []string, bool) {
	sources, skipped, ok := SelectAVLiveSources(parsed, requested)
	if len(sources) == 0 {
		return strings.ToLower(strings.TrimSpace(requested)), skipped, ok
	}
	return sources[0], skipped, ok
}

func SelectAVLiveSources(parsed AVNumber, requested string) ([]string, []string, bool) {
	source := strings.ToLower(strings.TrimSpace(requested))
	if source != "" && source != AVSourceAuto {
		return []string{source}, nil, IsImplementedAVLiveSource(source)
	}
	skipped := make([]string, 0)
	sources := make([]string, 0)
	seen := map[string]struct{}{}
	for _, provider := range parsed.PreferredProviders {
		provider = strings.ToLower(strings.TrimSpace(provider))
		if provider == "" {
			continue
		}
		if IsImplementedAVLiveSource(provider) {
			if _, ok := seen[provider]; !ok {
				seen[provider] = struct{}{}
				sources = append(sources, provider)
			}
			continue
		}
		skipped = append(skipped, provider)
	}
	if len(sources) == 0 {
		sources = append(sources, JavDBProvider)
	}
	return sources, skipped, true
}

func IsImplementedAVLiveSource(source string) bool {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case JavDBProvider, JavBusProvider:
		return true
	default:
		return false
	}
}
