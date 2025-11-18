package labgradient

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/png"
	"math"
)

// SliceParams describes a ring-based slice of CIE Lab space in the a/b plane.
// The slice lives on a circle of fixed Radius around the neutral gray point.
type SliceParams struct {
	L float64

	// Radius is the fixed distance from the neutral gray point in the a/b plane.
	Radius float64

	// CenterAngleDeg is the central hue angle in degrees
	// (0 = a>0, b=0; 90 = a=0, b>0; 180 = a<0, b=0; 270 = a=0, b<0).
	CenterAngleDeg float64

	// HorizontalSpanDeg controls how much the hue angle changes horizontally.
	HorizontalSpanDeg float64

	// VerticalSpanDeg controls additional angular modulation vertically.
	// This keeps the radius approximately constant while giving a 2D-looking gradient.
	VerticalSpanDeg float64
}

// GenerateLabSliceTexture builds a PNG texture for a Lab slice and returns it as a data URL.
// The slice is defined as a patch on the Lab chroma circle (constant Radius, varying angle) at lightness L.
func GenerateLabSliceTexture(width, height int, params SliceParams) (string, error) {
	if width <= 0 || height <= 0 {
		return "", ErrInvalidDimensions
	}

	if params.Radius <= 0 {
		return "", ErrInvalidRadius
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	centerRad := params.CenterAngleDeg * math.Pi / 180.0
	halfH := params.HorizontalSpanDeg * math.Pi / 180.0 / 2.0
	halfV := params.VerticalSpanDeg * math.Pi / 180.0 / 2.0

	for y := 0; y < height; y++ {
		// map to [-1, 1]
		v := 0.0
		if height > 1 {
			v = (float64(y)/float64(height-1))*2.0 - 1.0
		}

		for x := 0; x < width; x++ {
			u := 0.0
			if width > 1 {
				u = (float64(x)/float64(width-1))*2.0 - 1.0
			}

			// angle offset around the chroma circle, using both u and v to get a 2D patch
			angleOffset := u*halfH + v*halfV
			angle := centerRad + angleOffset

			a := params.Radius * math.Cos(angle)
			b := params.Radius * math.Sin(angle)

			r, g, bb := labToSRGB(params.L, a, b)
			img.Set(x, y, color.RGBA{
				R: r,
				G: g,
				B: bb,
				A: 255,
			})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}

	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	return "data:image/png;base64," + b64, nil
}

// GeneratePanoramicTexture generates a seamless 360-degree texture suitable for horizontal scrolling animation.
// width: Total width of the generated texture (should be large enough to hold the full spectrum at desired density).
// height: Height of the texture.
// params: L, Radius, VerticalSpanDeg are used. CenterAngleDeg and HorizontalSpanDeg are IGNORED (full 360 covered).
func GeneratePanoramicTexture(width, height int, params SliceParams) (string, error) {
	if width <= 0 || height <= 0 {
		return "", ErrInvalidDimensions
	}
	if params.Radius <= 0 {
		return "", ErrInvalidRadius
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// halfV adds the "twist" or vertical variation
	halfV := params.VerticalSpanDeg * math.Pi / 180.0 / 2.0

	for y := 0; y < height; y++ {
		// map y to [-1, 1] for the vertical twist
		v := 0.0
		if height > 1 {
			v = (float64(y)/float64(height-1))*2.0 - 1.0
		}

		for x := 0; x < width; x++ {
			// map x to [0, 1) -> [0, 2Pi)
			// seamless: x=0 is 0, x=width is effectively 2Pi (start of next cycle)
			normalizedX := float64(x) / float64(width)
			angle := normalizedX * 2.0 * math.Pi

			// Apply vertical twist
			// Note: we don't add CenterAngleDeg here because we cover the whole circle.
			// The CSS animation defines the "center" at any moment.
			angle += v * halfV

			a := params.Radius * math.Cos(angle)
			b := params.Radius * math.Sin(angle)

			r, g, bb := labToSRGB(params.L, a, b)
			img.Set(x, y, color.RGBA{
				R: r,
				G: g,
				B: bb,
				A: 255,
			})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}

	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	return "data:image/png;base64," + b64, nil
}

// ErrInvalidDimensions is returned when the requested texture size is non-positive.
var ErrInvalidDimensions = errors.New("texture dimensions must be positive")

// ErrInvalidRadius is returned when the requested Lab radius is non-positive.
var ErrInvalidRadius = errors.New("Lab radius must be positive")

// labToSRGB converts a single Lab point (D65) to sRGB with clipping.
func labToSRGB(L, a, b float64) (uint8, uint8, uint8) {
	// Lab (D65) -> XYZ
	const (
		epsilon = 216.0 / 24389.0
		kappa   = 24389.0 / 27.0
	)

	// D65 white point
	const (
		Xn = 95.047
		Yn = 100.0
		Zn = 108.883
	)

	fy := (L + 16.0) / 116.0
	fx := fy + a/500.0
	fz := fy - b/200.0

	fInv := func(t float64) float64 {
		t3 := t * t * t
		if t3 > epsilon {
			return t3
		}
		return (116.0*t - 16.0) / kappa
	}

	X := Xn * fInv(fx)
	Y := Yn * fInv(fy)
	Z := Zn * fInv(fz)

	// normalize to 0..1
	xr := X / 100.0
	yr := Y / 100.0
	zr := Z / 100.0

	// XYZ -> linear sRGB (D65)
	rLin := 3.2406*xr - 1.5372*yr - 0.4986*zr
	gLin := -0.9689*xr + 1.8758*yr + 0.0415*zr
	bLin := 0.0557*xr - 0.2040*yr + 1.0570*zr

	compand := func(c float64) float64 {
		if c <= 0.0 {
			return 0.0
		}
		if c >= 1.0 {
			return 1.0
		}
		if c <= 0.0031308 {
			return 12.92 * c
		}
		return 1.055*math.Pow(c, 1.0/2.4) - 0.055
	}

	r := compand(rLin)
	g := compand(gLin)
	bc := compand(bLin)

	to8 := func(c float64) uint8 {
		if c < 0 {
			c = 0
		}
		if c > 1 {
			c = 1
		}
		return uint8(math.Round(c * 255.0))
	}

	return to8(r), to8(g), to8(bc)
}
