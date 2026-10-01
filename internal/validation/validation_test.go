package validation

import (
	"strings"
	"testing"
)

func TestNormalizeUsername(t *testing.T) {
	tests := []struct{ in, want string }{
		{"  Kirasync2748  ", "kirasync2748"},
		{"GitHubUser", "githubuser"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := NormalizeUsername(tt.in); got != tt.want {
			t.Errorf("NormalizeUsername(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"kirasync2748", "kirasync2748", false},
		{"  Kirasync2748  ", "kirasync2748", false},
		{"user-name", "user-name", false},
		{"a", "a", false},
		{"a-b", "a-b", false},
		{"", "", true},
		{"   ", "", true},
		{"-leading", "", true},
		{"trailing-", "", true},
		{"has space", "", true},
		{"bad/slash", "", true},
		{"path..traversal", "", true},
		{"crlf\r\ninjection", "", true},
		{"tab\tchar", "", true},
		{"null\x00byte", "", true},
		{"emoji😀name", "", true},
		{"dot.in.name", "", true},
		{strings.Repeat("a", MaxUsernameLength+1), "", true},
		{"<script>", "", true},
	}
	for _, tt := range tests {
		got, err := ValidateUsername(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateUsername(%q) err = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("ValidateUsername(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRedisKey(t *testing.T) {
	if got := RedisKey("kirasync2748"); got != "pv:v1:user:kirasync2748" {
		t.Errorf("RedisKey = %q, want pv:v1:user:kirasync2748", got)
	}
}

func TestValidateColor(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"", DefaultColor, false},
		{"blue", "007ec6", false},
		{"BLUE", "007ec6", false},
		{"  green  ", "97ca00", false},
		{"red", "e05d44", false},
		{"dc143c", "dc143c", false},
		{"DC143C", "dc143c", false},
		{"#dc143c", "", true},
		{"dc143", "", true},
		{"dc143cc", "", true},
		{"gggggg", "", true},
		{"<svg>", "", true},
		{"blueviolet", "8a2be2", false},
		{"lightgrey", "9f9f9f", false},
	}
	for _, tt := range tests {
		got, err := ValidateColor(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateColor(%q) err = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("ValidateColor(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestValidateStyle(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"", DefaultStyle, false},
		{"flat", "flat", false},
		{"FLAT-SQUARE", "flat-square", false},
		{"  bold  ", "bold", false},
		{"  plastic  ", "plastic", false},
		{"capsule", "capsule", false},
		{"outline", "outline", false},
		{"for-the-badge", "for-the-badge", false},
		{"pixel", "pixel", false},
		{"unknown", "", true},
		{"<svg>", "", true},
	}
	for _, tt := range tests {
		got, err := ValidateStyle(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateStyle(%q) err = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("ValidateStyle(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestDefaultStyleIsBold(t *testing.T) {
	if DefaultStyle != "bold" {
		t.Errorf("DefaultStyle = %q, want bold", DefaultStyle)
	}
}

func TestStylesReturnsSeven(t *testing.T) {
	styles := Styles()
	if len(styles) != 7 {
		t.Errorf("Styles() returned %d styles, want 7", len(styles))
	}
	// for-the-badge alias should not be in the public list
	for _, s := range styles {
		if s == "for-the-badge" {
			t.Error("for-the-badge should not be in Styles() list")
		}
	}
}

func TestValidateLabel(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"", DefaultLabel, false},
		{"   ", DefaultLabel, false},
		{"PROFILE VIEWS", "PROFILE VIEWS", false},
		{"  spaced  ", "spaced", false},
		{"crlf\r\nlabel", "", true},
		{"tab\tlabel", "", true},
		{strings.Repeat("a", MaxLabelLength+1), "", true},
	}
	for _, tt := range tests {
		got, err := ValidateLabel(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateLabel(%q) err = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("ValidateLabel(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseBool(t *testing.T) {
	truthy := []string{"true", "TRUE", "1", "yes", "Yes", "on", "ON"}
	for _, v := range truthy {
		if !ParseBool(v) {
			t.Errorf("ParseBool(%q) = false, want true", v)
		}
	}
	falsy := []string{"", "false", "0", "no", "off", "anything"}
	for _, v := range falsy {
		if ParseBool(v) {
			t.Errorf("ParseBool(%q) = true, want false", v)
		}
	}
}
