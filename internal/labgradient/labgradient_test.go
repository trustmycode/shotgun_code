package labgradient

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"strings"
	"testing"
)

func decodeDataURLPNG(t *testing.T, dataURL string) (int, int) {
	t.Helper()
	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(dataURL, prefix) {
		t.Fatalf("unexpected data URL prefix: %q", dataURL)
	}
	encoded := strings.TrimPrefix(dataURL, prefix)
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("failed to decode base64: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("failed to decode png: %v", err)
	}
	bounds := img.Bounds()
	return bounds.Dx(), bounds.Dy()
}

func TestGenerateLabSliceTextureValidation(t *testing.T) {
	_, err := GenerateLabSliceTexture(0, 10, SliceParams{L: 80, Radius: 60})
	if err != ErrInvalidDimensions {
		t.Fatalf("expected ErrInvalidDimensions, got %v", err)
	}
	_, err = GenerateLabSliceTexture(10, 10, SliceParams{L: 80, Radius: 0})
	if err != ErrInvalidRadius {
		t.Fatalf("expected ErrInvalidRadius, got %v", err)
	}
}

func TestGeneratePanoramicTextureValidation(t *testing.T) {
	_, err := GeneratePanoramicTexture(-1, 10, SliceParams{L: 80, Radius: 60})
	if err != ErrInvalidDimensions {
		t.Fatalf("expected ErrInvalidDimensions, got %v", err)
	}
	_, err = GeneratePanoramicTexture(10, 10, SliceParams{L: 80, Radius: -1})
	if err != ErrInvalidRadius {
		t.Fatalf("expected ErrInvalidRadius, got %v", err)
	}
}

func TestGenerateLabSliceTextureProducesPNG(t *testing.T) {
	dataURL, err := GenerateLabSliceTexture(16, 8, SliceParams{
		L:                 80,
		Radius:            60,
		CenterAngleDeg:    180,
		HorizontalSpanDeg: 40,
		VerticalSpanDeg:   20,
	})
	if err != nil {
		t.Fatalf("GenerateLabSliceTexture failed: %v", err)
	}
	w, h := decodeDataURLPNG(t, dataURL)
	if w != 16 || h != 8 {
		t.Fatalf("unexpected png dimensions: %dx%d", w, h)
	}
}

func TestGeneratePanoramicTextureProducesPNG(t *testing.T) {
	dataURL, err := GeneratePanoramicTexture(32, 12, SliceParams{
		L:               80,
		Radius:          60,
		VerticalSpanDeg: 30,
	})
	if err != nil {
		t.Fatalf("GeneratePanoramicTexture failed: %v", err)
	}
	w, h := decodeDataURLPNG(t, dataURL)
	if w != 32 || h != 12 {
		t.Fatalf("unexpected png dimensions: %dx%d", w, h)
	}
}

func TestLabToSRGBBasicBounds(t *testing.T) {
	r, g, b := labToSRGB(100, 0, 0)
	if r < 240 || g < 240 || b < 240 {
		t.Fatalf("expected near white, got rgb(%d,%d,%d)", r, g, b)
	}

	r, g, b = labToSRGB(0, 0, 0)
	if r > 20 || g > 20 || b > 20 {
		t.Fatalf("expected near black, got rgb(%d,%d,%d)", r, g, b)
	}
}

func TestLabToSRGBClampExtremes(t *testing.T) {
	r, g, b := labToSRGB(90, 400, -400)
	if r > 255 || g > 255 || b > 255 {
		t.Fatalf("channels must be clamped to uint8 range, got rgb(%d,%d,%d)", r, g, b)
	}
}
