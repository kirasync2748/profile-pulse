package badge

import (
	"strings"
	"testing"
)

func TestEscape(t *testing.T) {
	tests := []struct{ in, want string }{
		{"hello", "hello"},
		{"a&b", "a&amp;b"},
		{"<tag>", "&lt;tag&gt;"},
		{`"q"`, "&quot;q&quot;"},
		{"it's", "it&#39;s"},
		{"ctrl\x01char", "ctrlchar"},
		{"<script>alert('x')</script>", "&lt;script&gt;alert(&#39;x&#39;)&lt;/script&gt;"},
	}
	for _, tt := range tests {
		if got := Escape(tt.in); got != tt.want {
			t.Errorf("Escape(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFormatNumber(t *testing.T) {
	if FormatNumber(1234) != "1234" {
		t.Errorf("FormatNumber(1234) = %q", FormatNumber(1234))
	}
	if FormatNumber(0) != "0" {
		t.Errorf("FormatNumber(0) = %q", FormatNumber(0))
	}
}

func TestAbbreviate(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1K"},
		{1200, "1.2K"},
		{12300, "12.3K"},
		{123000, "123K"},
		{1200000, "1.2M"},
		{12300000, "12.3M"},
		{1000000000, "1B"},
		{9999, "10K"},
		{1500, "1.5K"},
	}
	for _, tt := range tests {
		if got := Abbreviate(tt.in); got != tt.want {
			t.Errorf("Abbreviate(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestDisplayValue(t *testing.T) {
	if got := DisplayValue(250, 1000, false); got != "1250" {
		t.Errorf("DisplayValue(250,1000,false) = %q, want 1250", got)
	}
	// 1250 abbreviates to 1.3K (1 decimal place, round half away from zero).
	if got := DisplayValue(250, 1000, true); got != "1.3K" {
		t.Errorf("DisplayValue(250,1000,true) = %q, want 1.3K", got)
	}
	if got := DisplayValue(250, 0, false); got != "250" {
		t.Errorf("DisplayValue(250,0,false) = %q, want 250", got)
	}
}

func TestRenderFlat(t *testing.T) {
	svg := string(Render(NewParams("Profile views", "1234", "007ec6", "flat")))
	if !strings.HasPrefix(svg, "<svg") {
		t.Fatal("missing svg root")
	}
	if !strings.Contains(svg, `xmlns="http://www.w3.org/2000/svg"`) {
		t.Error("missing xmlns")
	}
	if !strings.Contains(svg, "Profile views") {
		t.Error("missing label")
	}
	if !strings.Contains(svg, "1234") {
		t.Error("missing value")
	}
	if !strings.Contains(svg, "#007ec6") {
		t.Error("missing color")
	}
	if !strings.HasSuffix(svg, "</svg>") {
		t.Error("missing closing tag")
	}
}

func TestRenderStyles(t *testing.T) {
	styles := []string{"flat", "flat-square", "bold", "plastic", "capsule", "outline", "pixel", "for-the-badge"}
	for _, s := range styles {
		svg := string(Render(NewParams("Profile views", "42", "blueviolet", s)))
		// pixel is a self-closing 1x1 svg; others end with </svg>.
		valid := strings.HasPrefix(svg, "<svg") && (strings.HasSuffix(svg, "</svg>") || strings.HasSuffix(svg, "/>"))
		if !valid {
			t.Errorf("style %s: invalid svg envelope", s)
		}
	}
}

func TestRenderPixelIsInvisible(t *testing.T) {
	svg := string(Render(NewParams("Profile views", "42", "007ec6", "pixel")))
	if !strings.Contains(svg, `width="1" height="1"`) {
		t.Errorf("pixel style should be 1x1, got %s", svg)
	}
}

func TestRenderEscapesDynamicText(t *testing.T) {
	svg := string(Render(NewParams("<script>", "</text>", "007ec6", "flat")))
	// The label must be escaped; the SVG markup itself contains </text>
	// closing tags, so we assert the escaped form of the value is present
	// rather than that the raw token is absent.
	if strings.Contains(svg, "<script>") {
		t.Error("label not escaped")
	}
	if !strings.Contains(svg, "&lt;script&gt;") {
		t.Error("label escaping missing")
	}
	if !strings.Contains(svg, "&lt;/text&gt;") {
		t.Error("value not escaped")
	}
}

func TestRenderBoldUppercases(t *testing.T) {
	svg := string(Render(NewParams("profile views", "42", "007ec6", "bold")))
	if !strings.Contains(svg, "PROFILE VIEWS") {
		t.Error("bold should uppercase the label")
	}
}

func TestRenderForTheBadgeAliasBold(t *testing.T) {
	bold := string(Render(NewParams("profile views", "42", "007ec6", "bold")))
	ftb := string(Render(NewParams("profile views", "42", "007ec6", "for-the-badge")))
	// Both should use the bold renderer: uppercase text, gradients, and sheen.
	for _, want := range []string{"PROFILE VIEWS", "linearGradient", "feDropShadow", "animation"} {
		if !strings.Contains(bold, want) {
			t.Errorf("bold output missing %q", want)
		}
		if !strings.Contains(ftb, want) {
			t.Errorf("for-the-badge output missing %q (should alias bold)", want)
		}
	}
}

func TestRenderBoldHasGradientAndShadow(t *testing.T) {
	svg := string(Render(NewParams("Profile views", "42", "0078d7", "bold")))
	if !strings.Contains(svg, "linearGradient") {
		t.Error("bold should contain gradient definitions")
	}
	if !strings.Contains(svg, "feDropShadow") {
		t.Error("bold should contain a drop shadow filter")
	}
	if !strings.Contains(svg, "animation") {
		t.Error("bold should contain a sheen animation")
	}
	if !strings.Contains(svg, "letter-spacing") {
		t.Error("bold should use letter-spacing")
	}
}

func TestRenderCapsuleRounded(t *testing.T) {
	svg := string(Render(NewParams("Profile views", "42", "007ec6", "capsule")))
	if !strings.Contains(svg, `rx="11"`) {
		t.Error("capsule should use rx=11 for pill shape")
	}
}

func TestRenderOutlineHasBorder(t *testing.T) {
	svg := string(Render(NewParams("Profile views", "42", "007ec6", "outline")))
	if !strings.Contains(svg, "stroke") {
		t.Error("outline should have a border stroke")
	}
}

func TestRenderValidXML(t *testing.T) {
	// Sanity: every style must contain a single self-closing svg or proper close.
	for _, s := range []string{"flat", "flat-square", "bold", "plastic", "capsule", "outline", "pixel", "for-the-badge"} {
		svg := string(Render(NewParams("Profile views", "1234", "007ec6", s)))
		if strings.Count(svg, "<svg") != 1 {
			t.Errorf("style %s: expected one svg root", s)
		}
	}
}
