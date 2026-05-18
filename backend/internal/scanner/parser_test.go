package scanner

import "testing"

func TestParseMovieVersion(t *testing.T) {
	parsed := ParseFile("/media/Movies/Inception.2010.2160p.BluRay.REMUX.HEVC.TrueHD.Atmos.HDR10-GROUP.mkv")

	if parsed.Title != "Inception" {
		t.Fatalf("expected title Inception, got %q", parsed.Title)
	}
	if parsed.Year != 2010 {
		t.Fatalf("expected year 2010, got %d", parsed.Year)
	}
	if parsed.Resolution != "2160p" {
		t.Fatalf("expected 2160p, got %q", parsed.Resolution)
	}
	if parsed.Source != "remux" {
		t.Fatalf("expected remux, got %q", parsed.Source)
	}
	if parsed.VideoCodec != "hevc" {
		t.Fatalf("expected hevc, got %q", parsed.VideoCodec)
	}
	if parsed.HDRFormat != "hdr10" {
		t.Fatalf("expected hdr10, got %q", parsed.HDRFormat)
	}
}

func TestParseTVEpisode(t *testing.T) {
	parsed := ParseFile("/media/TV/Show.Name.S02E03.1080p.WEB-DL.H264.CHS.mkv")

	if parsed.Title != "Show Name" {
		t.Fatalf("expected title Show Name, got %q", parsed.Title)
	}
	if parsed.Season != 2 || parsed.Episode != 3 {
		t.Fatalf("expected S02E03, got S%02dE%02d", parsed.Season, parsed.Episode)
	}
	if len(parsed.Subtitles) == 0 {
		t.Fatal("expected subtitle flags")
	}
}

func TestParseAVNumber(t *testing.T) {
	parsed := ParseFile("/media/AV/ABC-123-C.mp4")

	if parsed.Number != "ABC-123" {
		t.Fatalf("expected ABC-123, got %q", parsed.Number)
	}
	if parsed.Title != "ABC-123" {
		t.Fatalf("expected title fallback to number, got %q", parsed.Title)
	}
}

func TestParseFC2Number(t *testing.T) {
	parsed := ParseFile("/media/AV/FC2-PPV-1234567.mp4")

	if parsed.Number != "FC2-PPV-1234567" {
		t.Fatalf("expected FC2-PPV-1234567, got %q", parsed.Number)
	}
}
