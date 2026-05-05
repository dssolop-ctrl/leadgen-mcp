package wordstat

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestCompressWordstatResponse verifies the response size dropped from the
// raw 140 KB (8 phrases × heavy SearchedWith/SearchedAlso) down to a few KB.
func TestCompressWordstatResponse(t *testing.T) {
	// Build a synthetic merged response that mimics the real Wordstat shape:
	// 8 phrases, each with 50-item SearchedWith and SearchedAlso lists.
	items := make([]map[string]any, 8)
	for i := 0; i < 8; i++ {
		searched := make([]any, 50)
		also := make([]any, 50)
		for j := 0; j < 50; j++ {
			searched[j] = map[string]any{"Phrase": "купить квартиру вторичка модификатор " + string(rune('а'+j%30)), "Shows": 1000 - j*10}
			also[j] = map[string]any{"Phrase": "квартира вторичка похожая фраза " + string(rune('а'+j%30)), "Shows": 800 - j*5}
		}
		items[i] = map[string]any{
			"Phrase":       "тестовая фраза " + string(rune('1'+i)),
			"Shows":        12345 + i*100,
			"SearchedWith": searched,
			"SearchedAlso": also,
		}
	}
	rawBytes, _ := json.Marshal(items)
	rawSize := len(rawBytes)

	// Default mode (includeRelated=false) — must be much smaller than raw, no Searched* fields.
	compressed, err := compressWordstatResponse(string(rawBytes), false)
	if err != nil {
		t.Fatalf("compress (default) error: %v", err)
	}
	if len(compressed) >= rawSize/10 {
		t.Errorf("default compress: got %d bytes, want < %d (10%% of %d)", len(compressed), rawSize/10, rawSize)
	}
	if strings.Contains(compressed, "SearchedWith") || strings.Contains(compressed, "SearchedAlso") {
		t.Error("default compress should strip SearchedWith/SearchedAlso")
	}
	// Verify Phrase+Shows preserved.
	var compactItems []map[string]any
	if err := json.Unmarshal([]byte(compressed), &compactItems); err != nil {
		t.Fatalf("compressed not valid JSON: %v", err)
	}
	if len(compactItems) != 8 {
		t.Errorf("compressed length: got %d, want 8", len(compactItems))
	}
	for i, it := range compactItems {
		if _, ok := it["Phrase"]; !ok {
			t.Errorf("item %d missing Phrase", i)
		}
		if _, ok := it["Shows"]; !ok {
			t.Errorf("item %d missing Shows", i)
		}
	}

	// Extended mode (includeRelated=true) — keeps top-5 of each related list.
	extended, err := compressWordstatResponse(string(rawBytes), true)
	if err != nil {
		t.Fatalf("compress (extended) error: %v", err)
	}
	if !strings.Contains(extended, "SearchedWith") {
		t.Error("extended compress should keep SearchedWith")
	}
	var extItems []map[string]any
	if err := json.Unmarshal([]byte(extended), &extItems); err != nil {
		t.Fatalf("extended not valid JSON: %v", err)
	}
	for i, it := range extItems {
		if sw, ok := it["SearchedWith"].([]any); ok && len(sw) > 5 {
			t.Errorf("item %d SearchedWith length %d > 5", i, len(sw))
		}
		if sa, ok := it["SearchedAlso"].([]any); ok && len(sa) > 5 {
			t.Errorf("item %d SearchedAlso length %d > 5", i, len(sa))
		}
	}

	t.Logf("Sizes: raw=%d default=%d extended=%d (compression ratios: %.1fx default, %.1fx extended)",
		rawSize, len(compressed), len(extended),
		float64(rawSize)/float64(len(compressed)),
		float64(rawSize)/float64(len(extended)),
	)
}
