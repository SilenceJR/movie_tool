package scraper

import "testing"

func TestParseAVNumber(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		normalized string
		kind       string
		provider   string
	}{
		{name: "standard hyphen", input: "ABC-123", normalized: "ABC-123", kind: "standard", provider: "javdb"},
		{name: "standard compact", input: "ssni00123", normalized: "SSNI-00123", kind: "standard", provider: "javdb"},
		{name: "fc2 ppv", input: "FC2-PPV-1234567", normalized: "FC2-PPV-1234567", kind: "fc2", provider: "fc2"},
		{name: "fc2 compact", input: "fc2 1234567", normalized: "FC2-PPV-1234567", kind: "fc2", provider: "fc2"},
		{name: "heyzo", input: "HEYZO_1234", normalized: "HEYZO-1234", kind: "heyzo", provider: "heyzo"},
		{name: "carib", input: "carib-123456-789", normalized: "CARIB-123456-789", kind: "carib", provider: "carib"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, ok := ParseAVNumber(tt.input)
			if !ok {
				t.Fatalf("expected parse success for %q", tt.input)
			}
			if parsed.Normalized != tt.normalized || parsed.Kind != tt.kind {
				t.Fatalf("unexpected parse result: %#v", parsed)
			}
			if len(parsed.PreferredProviders) == 0 || parsed.PreferredProviders[0] != tt.provider {
				t.Fatalf("expected first provider %q, got %#v", tt.provider, parsed.PreferredProviders)
			}
		})
	}
}

func TestParseAVNumberRejectsUnknown(t *testing.T) {
	if parsed, ok := ParseAVNumber("holiday video"); ok {
		t.Fatalf("expected no parse, got %#v", parsed)
	}
}

func TestSelectAVLiveSourceAutoSkipsUnimplementedProviders(t *testing.T) {
	parsed, ok := ParseAVNumber("FC2-PPV-1234567")
	if !ok {
		t.Fatal("expected parse success")
	}
	sources, skipped, ok := SelectAVLiveSources(parsed, AVSourceAuto)
	if !ok || len(sources) != 2 || sources[0] != JavDBProvider || sources[1] != JavBusProvider {
		t.Fatalf("expected javdb/javbus fallback, got sources=%#v ok=%v", sources, ok)
	}
	if len(skipped) != 1 || skipped[0] != "fc2" {
		t.Fatalf("expected skipped fc2, got %#v", skipped)
	}
}

func TestSelectAVLiveSourceRejectsUnsupportedExplicitSource(t *testing.T) {
	source, _, ok := SelectAVLiveSource(AVNumber{}, "fc2")
	if ok || source != "fc2" {
		t.Fatalf("expected explicit fc2 to be rejected, got source=%q ok=%v", source, ok)
	}
}
