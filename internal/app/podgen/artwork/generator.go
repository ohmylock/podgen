package artwork

import (
	"crypto/sha256"
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/fogleman/gg"
)

// Style represents background style for artwork generation.
type Style string

const (
	StyleRandom           Style = ""                // Default (aurora)
	StyleSolid            Style = "solid"           // Solid color
	StyleGradientVertical Style = "gradient"        // Vertical gradient
	StyleGradientDiagonal Style = "gradient-diagonal"
	StyleRadial           Style = "radial"          // Radial gradient
	StyleCircles          Style = "circles"         // Soft circles
	StyleBlobs            Style = "blobs"           // Organic blob shapes
	StyleNoise            Style = "noise"           // Gradient with texture
	StyleLetter           Style = "letter"          // Big first letter
	StyleAurora           Style = "aurora"          // Mesh gradient (vibrant colors)
)

// AllStyles returns all available styles (excluding random).
func AllStyles() []Style {
	return []Style{
		StyleSolid,
		StyleGradientVertical,
		StyleGradientDiagonal,
		StyleRadial,
		StyleCircles,
		StyleBlobs,
		StyleNoise,
		StyleLetter,
		StyleAurora,
	}
}

const (
	imageSize    = 3000
	fontSize     = 180.0
	shadowOffset = 8.0
	textMaxWidth = 2400.0
)

var systemFontPaths = []string{
	// macOS
	"/Library/Fonts/Arial.ttf",
	"/System/Library/Fonts/Supplemental/Arial.ttf",
	"/System/Library/Fonts/Helvetica.ttc",
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

// Curated muted pastel palettes (designer-approved).
var palettes = [][]color.RGBA{
	// Palette 0: Sage & Dusty Blue
	{{189, 208, 196, 255}, {154, 183, 211, 255}, {232, 228, 225, 255}},
	// Palette 1: Warm Blush
	{{245, 210, 211, 255}, {247, 225, 211, 255}, {250, 248, 245, 255}},
	// Palette 2: Soft Lavender
	{{223, 204, 241, 255}, {200, 190, 220, 255}, {245, 243, 248, 255}},
	// Palette 3: Muted Teal
	{{163, 193, 218, 255}, {178, 227, 212, 255}, {240, 245, 245, 255}},
	// Palette 4: Warm Sand
	{{212, 196, 181, 255}, {200, 185, 175, 255}, {245, 242, 238, 255}},
}

// Generate creates a 3000x3000 PNG artwork image with the default style (aurora).
func Generate(seed, title, outputPath string) error {
	return GenerateWithStyle(seed, title, outputPath, StyleAurora)
}

// GenerateWithStyle creates artwork with a specific background style.
func GenerateWithStyle(seed, title, outputPath string, style Style) error {
	dc := gg.NewContext(imageSize, imageSize)

	// Default to aurora if no style specified
	if style == "" {
		style = StyleAurora
	}

	// Get colors from seed
	c1, c2, bg := colorsFromSeed(seed)

	// Draw background based on style
	switch style {
	case StyleSolid:
		drawSolid(dc, c1)
	case StyleGradientVertical:
		drawGradientVertical(dc, c1, c2)
	case StyleGradientDiagonal:
		drawGradientDiagonal(dc, c1, c2)
	case StyleRadial:
		drawRadial(dc, bg, c1)
	case StyleCircles:
		drawCircles(dc, seed, c1, c2, bg)
	case StyleBlobs:
		drawBlobs(dc, seed, c1, c2, bg)
	case StyleNoise:
		drawNoise(dc, seed, c1, c2)
	case StyleLetter:
		drawLetter(dc, seed, title, c1, c2, bg)
	case StyleAurora:
		drawAurora(dc, seed)
	default:
		drawGradientVertical(dc, c1, c2)
	}

	// Draw title text (except for letter style which handles it differently)
	if style != StyleLetter {
		// Aurora style: no shadow (dark background provides contrast)
		withShadow := style != StyleAurora
		if err := drawTitle(dc, title, withShadow); err != nil {
			return err
		}
	}

	if err := dc.SavePNG(outputPath); err != nil {
		return fmt.Errorf("artwork: save PNG %s: %w", outputPath, err)
	}
	return nil
}

// Background drawing functions

func drawSolid(dc *gg.Context, c1 color.RGBA) {
	dc.SetColor(c1)
	dc.Clear()
}

func drawGradientVertical(dc *gg.Context, c1, c2 color.RGBA) {
	grad := gg.NewLinearGradient(0, 0, 0, imageSize)
	grad.AddColorStop(0, c1)
	grad.AddColorStop(1, c2)
	dc.SetFillStyle(grad)
	dc.DrawRectangle(0, 0, imageSize, imageSize)
	dc.Fill()
}

func drawGradientDiagonal(dc *gg.Context, c1, c2 color.RGBA) {
	grad := gg.NewLinearGradient(0, 0, imageSize, imageSize)
	grad.AddColorStop(0, c1)
	grad.AddColorStop(1, c2)
	dc.SetFillStyle(grad)
	dc.DrawRectangle(0, 0, imageSize, imageSize)
	dc.Fill()
}

func drawRadial(dc *gg.Context, center, edge color.RGBA) {
	cx, cy := float64(imageSize)/2, float64(imageSize)/2
	maxR := float64(imageSize) * 0.9

	for y := 0; y < imageSize; y++ {
		for x := 0; x < imageSize; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			dist := math.Sqrt(dx*dx + dy*dy)
			t := math.Min(dist/maxR, 1.0)
			t = t * t * (3 - 2*t) // Smoothstep

			col := lerpColor(center, edge, t)
			dc.SetColor(col)
			dc.SetPixel(x, y)
		}
	}
}

func drawCircles(dc *gg.Context, seed string, c1, c2, bg color.RGBA) {
	dc.SetColor(bg)
	dc.Clear()

	// #nosec G404 - deterministic pseudo-random for artwork generation, not cryptographic
	rng := rand.New(rand.NewSource(seedToInt64(seed, 0)))

	for i := 0; i < 12; i++ {
		x := rng.Float64() * imageSize
		y := rng.Float64() * imageSize
		r := 400 + rng.Float64()*800

		var col color.RGBA
		if i%2 == 0 {
			// #nosec G115 - safe range: 25+[0,25) = [25,50)
			col = withAlpha(c1, uint8(25+rng.Intn(25)))
		} else {
			// #nosec G115 - safe range: 25+[0,25) = [25,50)
			col = withAlpha(c2, uint8(25+rng.Intn(25)))
		}
		dc.SetColor(col)
		dc.DrawCircle(x, y, r)
		dc.Fill()
	}
}

func drawBlobs(dc *gg.Context, seed string, c1, c2, bg color.RGBA) {
	dc.SetColor(bg)
	dc.Clear()

	// #nosec G404 - deterministic pseudo-random for artwork generation, not cryptographic
	rng := rand.New(rand.NewSource(seedToInt64(seed, 2)))
	colors := []color.RGBA{c1, c2}

	for i := 0; i < 5; i++ {
		cx := rng.Float64() * imageSize
		cy := rng.Float64() * imageSize
		baseR := 300 + rng.Float64()*500

		// #nosec G602,G115 - i%2 is always 0 or 1 (colors len=2), safe range: 40+[0,30) = [40,70)
		col := withAlpha(colors[i%2], uint8(40+rng.Intn(30)))
		dc.SetColor(col)

		dc.NewSubPath()
		points := 8
		for j := 0; j <= points; j++ {
			angle := float64(j) / float64(points) * 2 * math.Pi
			variation := 0.7 + rng.Float64()*0.6
			r := baseR * variation
			x := cx + math.Cos(angle)*r
			y := cy + math.Sin(angle)*r
			if j == 0 {
				dc.MoveTo(x, y)
			} else {
				dc.LineTo(x, y)
			}
		}
		dc.ClosePath()
		dc.Fill()
	}
}

func drawNoise(dc *gg.Context, seed string, c1, c2 color.RGBA) {
	// Base gradient
	drawGradientDiagonal(dc, c1, c2)

	// Add subtle noise overlay
	// #nosec G404 - deterministic pseudo-random for artwork generation, not cryptographic
	rng := rand.New(rand.NewSource(seedToInt64(seed, 3)))
	for y := 0; y < imageSize; y += 3 {
		for x := 0; x < imageSize; x += 3 {
			if rng.Float64() < 0.3 {
				// #nosec G115 - safe range: [0,20)
				noise := uint8(rng.Intn(20))
				if rng.Float64() < 0.5 {
					dc.SetColor(color.RGBA{255, 255, 255, noise})
				} else {
					dc.SetColor(color.RGBA{0, 0, 0, noise})
				}
				dc.DrawRectangle(float64(x), float64(y), 3, 3)
				dc.Fill()
			}
		}
	}
}

func drawAurora(dc *gg.Context, seed string) {
	// #nosec G404 - deterministic pseudo-random for artwork generation, not cryptographic
	rng := rand.New(rand.NewSource(seedToInt64(seed, 4)))

	// Generate vibrant aurora colors from seed
	auroraColors := generateAuroraColors(seed)

	// Dark base with slight color tint
	baseColor := darken(auroraColors[0], 0.85)
	dc.SetColor(baseColor)
	dc.Clear()

	// Create mesh gradient by placing multiple color "blobs"
	type colorBlob struct {
		x, y   float64
		radius float64
		color  color.RGBA
	}

	// Generate 4-6 blobs at positions AWAY from center (text zone)
	numBlobs := 4 + rng.Intn(3)
	blobs := make([]colorBlob, numBlobs)

	// Center exclusion zone for text
	centerX, centerY := float64(imageSize)/2, float64(imageSize)/2
	exclusionRadius := float64(imageSize) * 0.25

	for i := 0; i < numBlobs; i++ {
		// Place blobs in corners and edges, avoiding center
		var x, y float64
		for {
			x = rng.Float64() * imageSize
			y = rng.Float64() * imageSize
			// Check if far enough from center
			dx := x - centerX
			dy := y - centerY
			if math.Sqrt(dx*dx+dy*dy) > exclusionRadius {
				break
			}
		}
		blobs[i] = colorBlob{
			x:      x,
			y:      y,
			radius: float64(imageSize) * (0.3 + rng.Float64()*0.4),
			color:  auroraColors[i%len(auroraColors)],
		}
	}

	// Render mesh gradient pixel by pixel
	for y := 0; y < imageSize; y++ {
		for x := 0; x < imageSize; x++ {
			px, py := float64(x), float64(y)

			// Start with base color
			r, g, b := float64(baseColor.R), float64(baseColor.G), float64(baseColor.B)

			// Blend each blob's influence
			for _, blob := range blobs {
				dx := px - blob.x
				dy := py - blob.y
				dist := math.Sqrt(dx*dx + dy*dy)

				if dist < blob.radius {
					// Smooth falloff using smoothstep
					t := dist / blob.radius
					t = 1 - t*t*(3-2*t) // Inverse smoothstep for center brightness

					// Add color influence
					r += float64(blob.color.R) * t * 0.7
					g += float64(blob.color.G) * t * 0.7
					b += float64(blob.color.B) * t * 0.7
				}
			}

			// Clamp values
			r = math.Min(r, 255)
			g = math.Min(g, 255)
			b = math.Min(b, 255)

			dc.SetColor(color.RGBA{uint8(r), uint8(g), uint8(b), 255})
			dc.SetPixel(x, y)
		}
	}
}

// generateAuroraColors creates vibrant colors for aurora style based on seed.
func generateAuroraColors(seed string) []color.RGBA {
	h := sha256.Sum256([]byte(seed))

	// Generate 4 vibrant colors with different hues
	colors := make([]color.RGBA, 4)
	baseHue := float64(h[0]) / 255.0 * 360.0

	// Spread hues around the color wheel
	hues := []float64{
		baseHue,
		math.Mod(baseHue+90, 360),
		math.Mod(baseHue+180, 360),
		math.Mod(baseHue+270, 360),
	}

	for i, hue := range hues {
		// High saturation, varied lightness for vibrancy
		saturation := 0.8 + float64(h[i+1]%30)/100.0
		lightness := 0.45 + float64(h[i+2]%20)/100.0
		colors[i] = hslToRGBA(hue, saturation, lightness)
	}

	return colors
}

// hslToRGBA converts HSL values (h in 0-360, s and l in 0-1) to color.RGBA.
func hslToRGBA(h, s, l float64) color.RGBA {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}

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

// darken reduces the brightness of a color.
func darken(c color.RGBA, factor float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(c.R) * (1 - factor)),
		G: uint8(float64(c.G) * (1 - factor)),
		B: uint8(float64(c.B) * (1 - factor)),
		A: 255,
	}
}

func drawLetter(dc *gg.Context, seed, title string, c1, c2, bg color.RGBA) {
	// Soft gradient background
	grad := gg.NewLinearGradient(0, 0, imageSize, imageSize)
	grad.AddColorStop(0, bg)
	grad.AddColorStop(1, c1)
	dc.SetFillStyle(grad)
	dc.DrawRectangle(0, 0, imageSize, imageSize)
	dc.Fill()

	// Get first letter
	firstLetter := "?"
	for _, r := range title {
		firstLetter = string(r)
		break
	}

	// Draw big letter
	if loadFont(dc, 2000) {
		letterColor := withAlpha(c2, 80)
		dc.SetColor(letterColor)
		dc.DrawStringAnchored(firstLetter, imageSize/2, imageSize/2, 0.5, 0.5)
	}

	// Draw title on top
	_ = drawTitle(dc, title, true)
}

func drawTitle(dc *gg.Context, title string, withShadow bool) error {
	if !loadFont(dc, fontSize) {
		return fmt.Errorf("artwork: no system font found for text rendering")
	}

	// Shadow (optional)
	if withShadow {
		dc.SetColor(color.RGBA{0, 0, 0, 160})
		dc.DrawStringWrapped(title,
			imageSize/2+shadowOffset, imageSize/2+shadowOffset,
			0.5, 0.5, textMaxWidth, 1.4, gg.AlignCenter)
	}

	// Main text
	dc.SetColor(color.White)
	dc.DrawStringWrapped(title,
		imageSize/2, imageSize/2,
		0.5, 0.5, textMaxWidth, 1.4, gg.AlignCenter)

	return nil
}

// Helper functions

func loadFont(dc *gg.Context, points float64) bool {
	for _, p := range systemFontPaths {
		if err := dc.LoadFontFace(p, points); err == nil {
			return true
		}
	}
	return false
}

func colorsFromSeed(seed string) (c1, c2, bg color.RGBA) {
	h := sha256.Sum256([]byte(seed))
	paletteIdx := int(h[0]) % len(palettes)
	palette := palettes[paletteIdx]
	return palette[0], palette[1], palette[2]
}

func seedToInt64(seed string, offset int) int64 {
	h := sha256.Sum256([]byte(seed))
	return int64(h[offset%32])
}

func withAlpha(c color.RGBA, a uint8) color.RGBA {
	return color.RGBA{c.R, c.G, c.B, a}
}

func lerpColor(c1, c2 color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(c1.R)*(1-t) + float64(c2.R)*t),
		G: uint8(float64(c1.G)*(1-t) + float64(c2.G)*t),
		B: uint8(float64(c1.B)*(1-t) + float64(c2.B)*t),
		A: 255,
	}
}
