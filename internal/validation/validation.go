// Package validation sanitizes and validates untrusted query parameters used to
// build Redis keys and SVG badges.
//
// All user-supplied input is treated as hostile: usernames are constrained to a
// safe character set and bounded length before they ever touch a Redis key or
// SVG markup, and every value placed into SVG output is escaped by the badge
// package.
package validation

import (
	"fmt"
	"regexp"
	"strings"
)

// MaxUsernameLength is the maximum accepted GitHub username length. GitHub
// currently allows 39 characters; we allow a small margin while remaining
// well within safe bounds.
const MaxUsernameLength = 39

// usernameRe matches the characters permitted in a GitHub login: alphanumeric
// and hyphens. A GitHub username cannot start or end with a hyphen, but for
// counter-key purposes we only need a safe, normalized, injection-free key, so
// we accept the broader set and normalize.
var usernameRe = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?$`)

// NormalizeUsername lower-cases and trims a username. It does NOT validate;
// callers must run ValidateUsername on the result.
func NormalizeUsername(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

// ValidateUsername returns a normalized username or an error describing why
// the input is rejected. It guards against empty input, excessive length,
// illegal characters, control characters, path traversal and CRLF injection.
func ValidateUsername(raw string) (string, error) {
	u := NormalizeUsername(raw)
	if u == "" {
		return "", fmt.Errorf("username required")
	}
	if len(u) > MaxUsernameLength {
		return "", fmt.Errorf("invalid username: too long")
	}
	if strings.ContainsAny(u, "\r\n\t\x00") {
		return "", fmt.Errorf("invalid username")
	}
	if !usernameRe.MatchString(u) {
		return "", fmt.Errorf("invalid username")
	}
	return u, nil
}

// RedisKey builds the canonical Redis counter key for a validated username.
// The username must already be validated and normalized.
func RedisKey(username string) string {
	return "pv:v1:user:" + username
}

// Named colors supported by the badge renderer.
var namedColors = map[string]string{
	"brightgreen": "44cc11",
	"green":       "97ca00",
	"yellowgreen": "a4a61d",
	"yellow":      "dfb317",
	"orange":      "fe7d37",
	"red":         "e05d44",
	"blue":        "007ec6",
	"grey":        "555555",
	"lightgrey":   "9f9f9f",
	"blueviolet":  "8a2be2",
}

// DefaultColor is the badge color used when none is supplied.
const DefaultColor = "007ec6"

// hexColorRe matches a 6-digit hexadecimal color without a leading '#'.
var hexColorRe = regexp.MustCompile(`^[0-9a-fA-F]{6}$`)

// ValidateColor validates a color query parameter. Named colors resolve to
// their hex value. Bare hex colors (without '#') are accepted. Invalid input
// returns an error. The returned value is always a lowercase 6-digit hex
// string (without '#').
func ValidateColor(raw string) (string, error) {
	c := strings.ToLower(strings.TrimSpace(raw))
	if c == "" {
		return DefaultColor, nil
	}
	if named, ok := namedColors[c]; ok {
		return named, nil
	}
	if hexColorRe.MatchString(c) {
		return c, nil
	}
	return "", fmt.Errorf("invalid color")
}

// IsNamedColor reports whether raw is a supported named color.
func IsNamedColor(raw string) bool {
	_, ok := namedColors[strings.ToLower(strings.TrimSpace(raw))]
	return ok
}

// Supported badge styles.
const (
	StyleFlat        = "flat"
	StyleFlatSquare  = "flat-square"
	StyleBold        = "bold"
	StylePlastic     = "plastic"
	StyleCapsule     = "capsule"
	StyleOutline     = "outline"
	StylePixel       = "pixel"
	StyleForTheBadge = "for-the-badge" // alias for bold (backwards compatibility)
)

// DefaultStyle is the badge style used when none is supplied.
const DefaultStyle = StyleBold

var validStyles = map[string]bool{
	StyleFlat:        true,
	StyleFlatSquare:  true,
	StyleBold:        true,
	StylePlastic:     true,
	StyleCapsule:     true,
	StyleOutline:     true,
	StylePixel:       true,
	StyleForTheBadge: true,
}

// ValidateStyle validates a badge style query parameter.
func ValidateStyle(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return DefaultStyle, nil
	}
	if validStyles[s] {
		return s, nil
	}
	return "", fmt.Errorf("invalid style")
}

// Styles returns the list of supported badge styles shown to users. The
// for-the-badge alias is omitted; it is accepted for backwards compatibility
// but renders identically to bold.
func Styles() []string {
	return []string{StyleFlat, StyleFlatSquare, StyleBold, StylePlastic, StyleCapsule, StyleOutline, StylePixel}
}

// DefaultLabel is the badge label used when none is supplied.
const DefaultLabel = "Profile views"

// MaxLabelLength bounds the accepted label length to keep badges reasonable.
const MaxLabelLength = 64

// ValidateLabel validates and normalizes a badge label. It trims whitespace
// and enforces a maximum length. The returned label is the raw (still
// URL-decoded) text; SVG escaping is applied later by the badge package.
func ValidateLabel(raw string) (string, error) {
	l := strings.TrimSpace(raw)
	if l == "" {
		return DefaultLabel, nil
	}
	if len(l) > MaxLabelLength {
		return "", fmt.Errorf("invalid label: too long")
	}
	if strings.ContainsAny(l, "\r\n\t\x00") {
		return "", fmt.Errorf("invalid label")
	}
	return l, nil
}

// ParseBool parses a boolean-like query parameter (true/1/yes/on).
func ParseBool(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}
