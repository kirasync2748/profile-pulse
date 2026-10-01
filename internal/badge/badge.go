// Package badge generates SVG badge images for ProfilePulse.
//
// The renderer is an original implementation: it builds SVG markup in memory
// from Go string templates, measures label/value widths with a simple
// character-width heuristic, and escapes all dynamic text to prevent XML/SVG
// injection. It supports seven visually distinct styles and an invisible
// "pixel" counter style.
package badge

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync/atomic"
)

// Params describes a single badge to render.
type Params struct {
	Label string // pre-validated, will be XML-escaped
	Value string // pre-formatted display string, will be XML-escaped
	Color string // 6-digit hex without '#'
	Style string // validated style
}

// NewParams builds validated Params from already-validated inputs. Value is
// the display string (already formatted/abbreviated by the caller).
func NewParams(label, value, color, style string) Params {
	return Params{Label: label, Value: value, Color: color, Style: style}
}

// Escape escapes a string for safe inclusion in SVG/XML text content and
// attributes. It escapes the five mandatory XML characters and additionally
// neutralizes any remaining control characters.
func Escape(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 8)
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&#39;")
		default:
			if r < 0x20 && r != '\n' && r != '\t' {
				// drop control characters
				continue
			}
			b.WriteRune(r)
		}
	}
	return b.String()
}

// FormatNumber returns the raw integer string for the given count.
func FormatNumber(n int64) string {
	return strconv.FormatInt(n, 10)
}

// Abbreviate formats a count with K/M/B suffixes for compact display.
//
//	999      -> 999
//	1000     -> 1K
//	1200     -> 1.2K
//	12300    -> 12.3K
//	123000   -> 123K
//	1200000  -> 1.2M
//	12300000 -> 12.3M
//	1000000000 -> 1B
func Abbreviate(n int64) string {
	if n < 0 {
		n = 0
	}
	switch {
	case n >= 1_000_000_000:
		return formatScaled(n, 1_000_000_000, "B")
	case n >= 1_000_000:
		return formatScaled(n, 1_000_000, "M")
	case n >= 1_000:
		return formatScaled(n, 1_000, "K")
	default:
		return strconv.FormatInt(n, 10)
	}
}

func formatScaled(n, divisor int64, suffix string) string {
	f := float64(n) / float64(divisor)
	// Round to one decimal place.
	rounded := math.Round(f*10) / 10
	if rounded == float64(int64(rounded)) {
		return fmt.Sprintf("%d%s", int64(rounded), suffix)
	}
	return fmt.Sprintf("%.1f%s", rounded, suffix)
}

// DisplayValue returns the value string for a badge given the raw count, the
// base offset, and the abbreviation flag.
func DisplayValue(count, base int64, abbreviated bool) string {
	total := count + base
	if total < 0 {
		total = 0
	}
	if abbreviated {
		return Abbreviate(total)
	}
	return FormatNumber(total)
}

// --- color helpers ----------------------------------------------------------

// parseHex parses a 6-digit hex color string (without '#') into RGB components.
func parseHex(hex string) (r, g, b int, ok bool) {
	if len(hex) != 6 {
		return 0, 0, 0, false
	}
	_, err := fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	if err != nil {
		return 0, 0, 0, false
	}
	return r, g, b, true
}

// lightenHex returns a hex color blended toward white by the given amount
// (0 = unchanged, 1 = white). If the input is not a valid 6-digit hex, it is
// returned unchanged.
func lightenHex(hex string, amount float64) string {
	r, g, b, ok := parseHex(hex)
	if !ok {
		return hex
	}
	blend := func(v int) int {
		return int(math.Round(float64(v) + (255-float64(v))*amount))
	}
	return fmt.Sprintf("%02x%02x%02x", blend(r), blend(g), blend(b))
}

// --- unique IDs -------------------------------------------------------------

var idCounter uint64

// uniqueID returns a short unique string for SVG element IDs, so multiple
// badges on the same page never collide.
func uniqueID() string {
	return strconv.FormatUint(atomic.AddUint64(&idCounter, 1), 36)
}

// --- text measurement -------------------------------------------------------

// charWidth approximates the advance width of a character in the badge font at
// 11px (the de-facto badge font size). This is a deliberate, original
// heuristic; it does not replicate any other project's measurements.
const (
	fontSize      = 11
	charWidth     = 6.6 // average advance width at 11px for the system sans font
	charWidthBold = 7.0
	padding       = 5 // horizontal padding inside a segment
	gap           = 0 // gap between segments (flat styles have none)
)

func textWidth(s string, w float64) float64 {
	return float64(len(s)) * w
}

// segmentWidth returns the inner width (text width) plus padding for a segment.
func segmentWidth(text string, w float64) float64 {
	return textWidth(text, w) + 2*padding
}

// layout measures the two segments and returns rounded pixel widths plus the
// text anchor positions. The value segment is always extended to the right
// edge (valueR = totalW - labelR) so the rounded right corner is filled by the
// value color with no gap, and both texts are centered on their segment's
// midpoint.
func layout(labelW, valueW float64, minW int) (labelR, valueR, totalW, labelX, valueX int) {
	labelR = int(math.Round(labelW))
	valueR = int(math.Round(valueW))
	totalW = labelR + valueR
	if totalW < minW {
		totalW = minW
	}
	valueR = totalW - labelR // fill to the right edge — no gap after the number
	labelX = int(math.Round(float64(labelR) / 2))
	valueX = labelR + int(math.Round(float64(valueR)/2))
	return
}

// --- rendering --------------------------------------------------------------

// Render produces the SVG bytes for the given params.
func Render(p Params) []byte {
	switch p.Style {
	case "flat-square":
		return renderFlatSquare(p)
	case "bold", "for-the-badge": // for-the-badge is a backwards-compatible alias
		return renderBold(p)
	case "plastic":
		return renderPlastic(p)
	case "capsule":
		return renderCapsule(p)
	case "outline":
		return renderOutline(p)
	case "pixel":
		return renderPixel(p)
	default:
		return renderFlat(p)
	}
}

// renderFlat is the default flat badge with rounded right corners.
func renderFlat(p Params) []byte {
	labelR, valueR, totalW, labelX, valueX := layout(
		segmentWidth(p.Label, charWidth),
		segmentWidth(p.Value, charWidthBold),
		20,
	)
	color := p.Color
	if color == "" {
		color = "007ec6"
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20" role="img" aria-label="%s: %s">`, totalW, Escape(p.Label), Escape(p.Value))
	b.WriteString(`<linearGradient id="s" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient>`)
	b.WriteString(`<clipPath id="r"><rect width="` + itoa(totalW) + `" height="20" rx="3"/></clipPath>`)
	b.WriteString(`<g clip-path="url(#r)"><rect width="` + itoa(labelR) + `" height="20" fill="#555"/><rect x="` + itoa(labelR) + `" width="` + itoa(valueR) + `" height="20" fill="#` + color + `"/><rect width="` + itoa(totalW) + `" height="20" fill="url(#s)"/></g>`)
	b.WriteString(`<g fill="#fff" text-anchor="middle" font-family="Verdana,DejaVu Sans,sans-serif" font-size="11" text-rendering="geometricPrecision">`)
	fmt.Fprintf(&b, `<text x="%d" y="15" fill="#010101" fill-opacity=".3">%s</text>`, labelX, Escape(p.Label))
	fmt.Fprintf(&b, `<text x="%d" y="14">%s</text>`, labelX, Escape(p.Label))
	fmt.Fprintf(&b, `<text x="%d" y="15" fill="#010101" fill-opacity=".3">%s</text>`, valueX, Escape(p.Value))
	fmt.Fprintf(&b, `<text x="%d" y="14">%s</text>`, valueX, Escape(p.Value))
	b.WriteString(`</g></svg>`)
	return []byte(b.String())
}

// renderFlatSquare is a flat badge with square corners.
func renderFlatSquare(p Params) []byte {
	labelR, valueR, totalW, labelX, valueX := layout(
		segmentWidth(p.Label, charWidth),
		segmentWidth(p.Value, charWidthBold),
		20,
	)
	color := p.Color
	if color == "" {
		color = "007ec6"
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20" role="img" aria-label="%s: %s">`, totalW, Escape(p.Label), Escape(p.Value))
	fmt.Fprintf(&b, `<rect width="%d" height="20" fill="#555"/>`, labelR)
	fmt.Fprintf(&b, `<rect x="%d" width="%d" height="20" fill="#%s"/>`, labelR, valueR, color)
	b.WriteString(`<g fill="#fff" text-anchor="middle" font-family="Verdana,DejaVu Sans,sans-serif" font-size="11" text-rendering="geometricPrecision">`)
	fmt.Fprintf(&b, `<text x="%d" y="14">%s</text>`, labelX, Escape(p.Label))
	fmt.Fprintf(&b, `<text x="%d" y="14">%s</text>`, valueX, Escape(p.Value))
	b.WriteString(`</g></svg>`)
	return []byte(b.String())
}

// renderPlastic is a glossy badge with rounded corners and a gradient sheen.
func renderPlastic(p Params) []byte {
	const (
		plasticHeight = 18
		plasticRadius = 4
	)
	labelR, valueR, totalW, labelX, valueX := layout(
		textWidth(p.Label, charWidth)+2*padding,
		textWidth(p.Value, charWidthBold)+2*padding,
		20,
	)
	color := p.Color
	if color == "" {
		color = "007ec6"
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" role="img" aria-label="%s: %s">`, totalW, plasticHeight, Escape(p.Label), Escape(p.Value))
	b.WriteString(`<linearGradient id="g" x2="0" y2="100%"><stop offset="0" stop-color="#fff" stop-opacity=".7"/><stop offset=".5" stop-color="#fff" stop-opacity="0"/><stop offset="1" stop-color="#000" stop-opacity=".1"/></linearGradient>`)
	fmt.Fprintf(&b, `<clipPath id="r"><rect width="%d" height="%d" rx="%d"/></clipPath>`, totalW, plasticHeight, plasticRadius)
	fmt.Fprintf(&b, `<g clip-path="url(#r)"><rect width="%d" height="%d" fill="#555"/><rect x="%d" width="%d" height="%d" fill="#%s"/><rect width="%d" height="%d" fill="url(#g)"/></g>`, labelR, plasticHeight, labelR, valueR, plasticHeight, color, totalW, plasticHeight)
	b.WriteString(`<g fill="#fff" text-anchor="middle" font-family="Verdana,DejaVu Sans,sans-serif" font-size="11" text-rendering="geometricPrecision">`)
	fmt.Fprintf(&b, `<text x="%d" y="13" fill="#010101" fill-opacity=".3">%s</text>`, labelX, Escape(p.Label))
	fmt.Fprintf(&b, `<text x="%d" y="12">%s</text>`, labelX, Escape(p.Label))
	fmt.Fprintf(&b, `<text x="%d" y="13" fill="#010101" fill-opacity=".3">%s</text>`, valueX, Escape(p.Value))
	fmt.Fprintf(&b, `<text x="%d" y="12">%s</text>`, valueX, Escape(p.Value))
	b.WriteString(`</g></svg>`)
	return []byte(b.String())
}

// renderBold is a large, bold badge with uppercase text, gradient segments, a
// drop shadow, inner top highlight, and a sweeping sheen animation. This is the
// default style.
func renderBold(p Params) []byte {
	const (
		boldCharW   = 7.55 // bold weight + 0.55px letter-spacing
		boldPadding = 13.0
		boldHeight  = 28
		boldRadius  = 4
	)

	label := strings.ToUpper(p.Label)
	value := strings.ToUpper(p.Value)

	labelR, valueR, totalW, labelX, valueX := layout(
		textWidth(label, boldCharW)+2*boldPadding,
		textWidth(value, boldCharW)+2*boldPadding,
		40,
	)

	color := p.Color
	if color == "" {
		color = "0078d7"
	}
	lightColor := lightenHex(color, 0.35)
	uid := uniqueID()

	// Sheen sweep geometry (matches the approved CSS design).
	sheenW := 0.35 * float64(totalW)
	sheenStartX := -0.45 * float64(totalW)
	sheenTranslate := 4.7 * sheenW

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" role="img" aria-label="%s: %s">`, totalW, boldHeight, Escape(p.Label), Escape(p.Value))

	// Definitions: drop-shadow filter, segment gradients, sheen gradient, clip path.
	b.WriteString(`<defs>`)
	fmt.Fprintf(&b, `<filter id="ds%s" x="-15%%" y="-15%%" width="130%%" height="160%%"><feDropShadow dx="0" dy="2" stdDeviation="2.5" flood-color="rgb(15,35,55)" flood-opacity="0.24"/></filter>`, uid)
	fmt.Fprintf(&b, `<linearGradient id="lg%s" x2="0" y2="1"><stop offset="0" stop-color="#20374e"/><stop offset="1" stop-color="#16283b"/></linearGradient>`, uid)
	fmt.Fprintf(&b, `<linearGradient id="vg%s" x2="0" y2="1"><stop offset="0" stop-color="#%s"/><stop offset="1" stop-color="#%s"/></linearGradient>`, uid, lightColor, color)
	fmt.Fprintf(&b, `<linearGradient id="sh%s" x2="1" y2="0"><stop offset="0" stop-color="#fff" stop-opacity="0"/><stop offset="0.5" stop-color="#fff" stop-opacity="0.23"/><stop offset="1" stop-color="#fff" stop-opacity="0"/></linearGradient>`, uid)
	fmt.Fprintf(&b, `<clipPath id="cp%s"><rect width="%d" height="%d" rx="%d"/></clipPath>`, uid, totalW, boldHeight, boldRadius)
	b.WriteString(`</defs>`)

	// Segments + inner highlight — filter wraps the clipped group so the drop
	// shadow renders around the rounded shape (filter applies before clip).
	fmt.Fprintf(&b, `<g filter="url(#ds%s)"><g clip-path="url(#cp%s)">`, uid, uid)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="#16283b"/>`, totalW, boldHeight)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="url(#lg%s)"/>`, labelR, boldHeight, uid)
	fmt.Fprintf(&b, `<rect x="%d" width="%d" height="%d" fill="url(#vg%s)"/>`, labelR, valueR, boldHeight, uid)
	fmt.Fprintf(&b, `<rect width="%d" height="1" fill="#fff" opacity="0.18"/>`, totalW) // inset top highlight
	b.WriteString(`</g></g>`)

	// Sheen sweep — clipped to the badge, animated via CSS keyframes.
	fmt.Fprintf(&b, `<style>.sw%s{animation:sw%s 4.8s ease-in-out infinite}@keyframes sw%s{0%%,62%%{transform:translate(0px,0);opacity:0}70%%{opacity:1}100%%{transform:translate(%.0fpx,0);opacity:0}}</style>`, uid, uid, uid, sheenTranslate)
	fmt.Fprintf(&b, `<g clip-path="url(#cp%s)"><rect class="sw%s" x="%.0f" width="%.0f" height="%d" fill="url(#sh%s)"/></g>`, uid, uid, sheenStartX, sheenW, boldHeight, uid)

	// Text — label has no shadow; value has a 1px-down dark shadow.
	b.WriteString(`<g fill="#fff" text-anchor="middle" font-family="Arial,Helvetica,sans-serif" font-size="11" font-weight="800" letter-spacing="0.55" text-rendering="geometricPrecision">`)
	fmt.Fprintf(&b, `<text x="%d" y="19">%s</text>`, labelX, Escape(label))
	fmt.Fprintf(&b, `<text x="%d" y="20" fill="#00285a" fill-opacity="0.3">%s</text>`, valueX, Escape(value))
	fmt.Fprintf(&b, `<text x="%d" y="19">%s</text>`, valueX, Escape(value))
	b.WriteString(`</g></svg>`)
	return []byte(b.String())
}

// renderCapsule is a fully rounded pill-shaped flat badge.
func renderCapsule(p Params) []byte {
	const (
		capsuleCharW     = 6.6
		capsuleCharWBold = 7.0
		capsulePadding   = 7.0
		capsuleHeight    = 22
		capsuleRadius    = 11
	)

	labelR, valueR, totalW, labelX, valueX := layout(
		textWidth(p.Label, capsuleCharW)+2*capsulePadding,
		textWidth(p.Value, capsuleCharWBold)+2*capsulePadding,
		30,
	)
	color := p.Color
	if color == "" {
		color = "007ec6"
	}
	uid := uniqueID()

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" role="img" aria-label="%s: %s">`, totalW, capsuleHeight, Escape(p.Label), Escape(p.Value))
	fmt.Fprintf(&b, `<clipPath id="cp%s"><rect width="%d" height="%d" rx="%d"/></clipPath>`, uid, totalW, capsuleHeight, capsuleRadius)
	fmt.Fprintf(&b, `<g clip-path="url(#cp%s)">`, uid)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="#555"/>`, labelR, capsuleHeight)
	fmt.Fprintf(&b, `<rect x="%d" width="%d" height="%d" fill="#%s"/>`, labelR, valueR, capsuleHeight, color)
	b.WriteString(`</g>`)
	b.WriteString(`<g fill="#fff" text-anchor="middle" font-family="Verdana,DejaVu Sans,sans-serif" font-size="11" text-rendering="geometricPrecision">`)
	fmt.Fprintf(&b, `<text x="%d" y="16" fill="#010101" fill-opacity=".3">%s</text>`, labelX, Escape(p.Label))
	fmt.Fprintf(&b, `<text x="%d" y="15">%s</text>`, labelX, Escape(p.Label))
	fmt.Fprintf(&b, `<text x="%d" y="16" fill="#010101" fill-opacity=".3">%s</text>`, valueX, Escape(p.Value))
	fmt.Fprintf(&b, `<text x="%d" y="15">%s</text>`, valueX, Escape(p.Value))
	b.WriteString(`</g></svg>`)
	return []byte(b.String())
}

// renderOutline is a minimal badge with a 1px border and flat colors.
func renderOutline(p Params) []byte {
	const (
		outlineCharW     = 6.6
		outlineCharWBold = 7.0
		outlinePadding   = 6.0
		outlineHeight    = 20
		outlineRadius    = 3
	)

	labelR, valueR, totalW, labelX, valueX := layout(
		textWidth(p.Label, outlineCharW)+2*outlinePadding,
		textWidth(p.Value, outlineCharWBold)+2*outlinePadding,
		30,
	)
	color := p.Color
	if color == "" {
		color = "007ec6"
	}
	uid := uniqueID()

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" role="img" aria-label="%s: %s">`, totalW, outlineHeight, Escape(p.Label), Escape(p.Value))
	fmt.Fprintf(&b, `<clipPath id="cp%s"><rect width="%d" height="%d" rx="%d"/></clipPath>`, uid, totalW, outlineHeight, outlineRadius)
	fmt.Fprintf(&b, `<g clip-path="url(#cp%s)">`, uid)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="#6b6b6b"/>`, labelR, outlineHeight)
	fmt.Fprintf(&b, `<rect x="%d" width="%d" height="%d" fill="#%s"/>`, labelR, valueR, outlineHeight, color)
	b.WriteString(`</g>`)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" rx="%d" fill="none" stroke="#cbd5dc" stroke-width="1"/>`, totalW-1, outlineHeight-1, outlineRadius)
	b.WriteString(`<g fill="#fff" text-anchor="middle" font-family="Verdana,DejaVu Sans,sans-serif" font-size="11" text-rendering="geometricPrecision">`)
	fmt.Fprintf(&b, `<text x="%d" y="14">%s</text>`, labelX, Escape(p.Label))
	fmt.Fprintf(&b, `<text x="%d" y="14" font-weight="bold">%s</text>`, valueX, Escape(p.Value))
	b.WriteString(`</g></svg>`)
	return []byte(b.String())
}

// renderPixel is an invisible 1x1 counter image. It returns a minimal
// transparent SVG so the request is still counted without displaying a badge.
func renderPixel(p Params) []byte {
	return []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="1" height="1"/>`)
}

// itoa is a tiny strconv-free int to string helper to keep the hot path
// allocation-light.
func itoa(n int) string { return strconv.Itoa(n) }
