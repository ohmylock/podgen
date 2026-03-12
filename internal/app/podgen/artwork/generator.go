package artwork

import (
	"crypto/sha256"
	"fmt"
	"image/color"
	"math"

	"github.com/fogleman/gg"
)

const (
	imageSize    = 3000
	fontSize     = 180.0
	shadowOffset = 8.0
	textMaxWidth = 2400.0
)

// systemFontPaths lists common TTF font paths on macOS, Linux, and Windows.
var systemFontPaths = []string{
	// macOS
	"/Library/Fonts/Arial.ttf",
	"/System/Library/Fonts/Supplemental/Arial.ttf",
	// Linux
	"/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
	"/usr/share/fonts/truetype/liberation/LiberationSans-Bold.ttf",
	"/usr/share/fonts/truetype/ubuntu/Ubuntu-B.ttf",
	"/usr/share/fonts/truetype/freefont/FreeSansBold.ttf",
	// Windows
	"C:\\Windows\\Fonts\\arial.ttf",
	"C:\\Windows\\Fonts\\Arial.ttf",
	"C:\\Windows\\Fonts\\segoeui.ttf",
}

// Generate creates a 3000x3000 PNG artwork image with a vertical gradient background
// and centered title text. The gradient colors are derived deterministically from seed.
func Generate(seed, title, outputPath string) error {
	c1, c2 := colorsFromSeed(seed)

	dc := gg.NewContext(imageSize, imageSize)

	// Draw vertical gradient background
	grad := gg.NewLinearGradient(0, 0, 0, imageSize)
	grad.AddColorStop(0, c1)
	grad.AddColorStop(1, c2)
	dc.SetFillStyle(grad)
	dc.DrawRectangle(0, 0, imageSize, imageSize)
	dc.Fill()

	// Try to load a system font; error if none found (gg has no built-in font)
	if !loadFont(dc, fontSize) {
		return fmt.Errorf("artwork: no system font found for text rendering")
	}

	// Draw text shadow
	dc.SetColor(color.RGBA{R: 0, G: 0, B: 0, A: 160})
	dc.DrawStringWrapped(title,
		imageSize/2+shadowOffset, imageSize/2+shadowOffset,
		0.5, 0.5,
		textMaxWidth, 1.4,
		gg.AlignCenter)

	// Draw main text in white
	dc.SetColor(color.White)
	dc.DrawStringWrapped(title,
		imageSize/2, imageSize/2,
		0.5, 0.5,
		textMaxWidth, 1.4,
		gg.AlignCenter)

	if err := dc.SavePNG(outputPath); err != nil {
		return fmt.Errorf("artwork: save PNG %s: %w", outputPath, err)
	}
	return nil
}

// loadFont tries system font paths in order and loads the first one found.
// Returns true if a font was loaded; the built-in gg font is used as fallback.
func loadFont(dc *gg.Context, points float64) bool {
	for _, p := range systemFontPaths {
		if err := dc.LoadFontFace(p, points); err == nil {
			return true
		}
	}
	return false
}

// colorsFromSeed generates two visually pleasing gradient colors from a seed string.
// Colors are derived deterministically via SHA-256 using HSL color space.
func colorsFromSeed(seed string) (top, bottom color.Color) {
	h := sha256.Sum256([]byte(seed))

	// Map first hash byte to a hue (0–359°), use split-complementary for second
	hue1 := float64(h[0]) / 255.0 * 360.0
	hue2 := math.Mod(hue1+150, 360)

	// High saturation, mid-range lightness for vivid but not garish gradients
	c1 := hslToRGBA(hue1, 0.65, 0.45)
	c2 := hslToRGBA(hue2, 0.65, 0.35)
	return c1, c2
}

// hslToRGBA converts HSL values (h in 0–360, s and l in 0–1) to color.RGBA.
func hslToRGBA(h, s, l float64) color.RGBA {
	h = math.Mod(h, 360)
	s = clamp01(s)
	l = clamp01(l)

	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2

	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}

	return color.RGBA{
		R: uint8(math.Round((r + m) * 255)),
		G: uint8(math.Round((g + m) * 255)),
		B: uint8(math.Round((b + m) * 255)),
		A: 255,
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
