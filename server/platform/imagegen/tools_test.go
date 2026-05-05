package imagegen

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"testing"
)

// makePNG returns a synthetic w×h PNG with a deterministic colour pattern.
func makePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 255), uint8(y % 255), 128, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.Bytes()
}

// TestCenterCropToAspect verifies the auto-crop produces correct ratios for
// every aspect we support, given off-ratio source images. This is the backstop
// for Flux / Gemini ignoring image_size.
func TestCenterCropToAspect(t *testing.T) {
	cases := []struct {
		name           string
		srcW, srcH     int
		aspect         string
		expW, expH     int
		ratioTolerance float64 // relative
	}{
		// 4:3 source (1024x768) → 1:1 should crop to 768x768.
		{"4:3 → 1:1", 1024, 768, "1:1", 768, 768, 0.001},
		// 4:3 source → 16:9 should crop to 1024x576 (1024/576 = 1.778).
		{"4:3 → 16:9", 1024, 768, "16:9", 1024, 576, 0.005},
		// 1:1 source → 16:9 should crop to 1024x576.
		{"1:1 → 16:9", 1024, 1024, "16:9", 1024, 576, 0.005},
		// 16:9 source → 1:1 should crop to 576x576.
		{"16:9 → 1:1", 1024, 576, "1:1", 576, 576, 0.001},
		// Already correct: 1024x1024 → 1:1 — function should be no-op-equivalent
		// (returns same dims; called only when checkAspectMatch flagged it,
		// so in production we skip; here we test the crop itself).
		{"1:1 → 1:1 same", 800, 800, "1:1", 800, 800, 0.001},
		// 9:16 portrait source → 9:16 same.
		{"9:16 → 9:16 same", 720, 1280, "9:16", 720, 1280, 0.001},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := makePNG(t, c.srcW, c.srcH)
			out, w, h, err := centerCropToAspect(src, c.aspect, "image/png")
			if err != nil {
				t.Fatalf("crop failed: %v", err)
			}
			if w != c.expW || h != c.expH {
				t.Errorf("got %dx%d, expected %dx%d", w, h, c.expW, c.expH)
			}
			// Verify encoded bytes decode and have the expected dimensions.
			img, _, derr := image.Decode(bytes.NewReader(out))
			if derr != nil {
				t.Fatalf("decode result: %v", derr)
			}
			b := img.Bounds()
			if b.Dx() != c.expW || b.Dy() != c.expH {
				t.Errorf("decoded %dx%d, expected %dx%d", b.Dx(), b.Dy(), c.expW, c.expH)
			}
			// Verify ratio is correct within tolerance.
			want, _ := aspectRatioFloat(c.aspect)
			got := float64(w) / float64(h)
			if math.Abs(got-want)/want > c.ratioTolerance {
				t.Errorf("ratio %.4f, want %.4f (±%.3f)", got, want, c.ratioTolerance)
			}
		})
	}
}

// TestCheckAspectMatch verifies the warning is emitted when (and only when)
// the actual ratio diverges from the requested by more than 2%.
func TestCheckAspectMatch(t *testing.T) {
	cases := []struct {
		aspect    string
		w, h      int
		wantWarn  bool
	}{
		{"1:1", 1024, 1024, false},
		{"1:1", 1024, 768, true},   // 4:3 ≠ 1:1
		{"16:9", 1920, 1080, false},
		{"16:9", 1376, 768, false}, // 1.792 vs 1.778, within 2%
		{"16:9", 1024, 768, true},  // 1.333 way off
		{"4:3", 1024, 768, false},
		{"4:3", 1024, 1024, true},
	}
	for _, c := range cases {
		got := checkAspectMatch(c.aspect, c.w, c.h)
		if (got != "") != c.wantWarn {
			t.Errorf("aspect=%s %dx%d: warn=%q, wantWarn=%v", c.aspect, c.w, c.h, got, c.wantWarn)
		}
	}
}

// TestAspectRatioFloat is a basic sanity check on the parser.
func TestAspectRatioFloat(t *testing.T) {
	cases := map[string]float64{
		"":     1.0,
		"1:1":  1.0,
		"16:9": 16.0 / 9.0,
		"4:3":  4.0 / 3.0,
		"3:2":  1.5,
		"9:16": 9.0 / 16.0,
	}
	for aspect, want := range cases {
		got, ok := aspectRatioFloat(aspect)
		if !ok || math.Abs(got-want) > 0.0001 {
			t.Errorf("aspect=%q got=%v ok=%v want=%v", aspect, got, ok, want)
		}
	}
	if _, ok := aspectRatioFloat("unknown"); ok {
		t.Error("unknown aspect should return ok=false")
	}
}
