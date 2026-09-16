package create_test

import (
	"strings"
	"testing"

	"wptui/internal/create"
)

func TestSlugify(t *testing.T) {
	longName := strings.TrimSpace(strings.Repeat("long ", 20))
	expectedLongSlug := strings.Repeat("long-", 19) + "long"

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple english", "My Website", "my-website"},
		{"vietnamese accented", "Cửa Hàng Mỹ Phẩm", "cua-hang-my-pham"},
		{"special characters", "Special! @# $% Tests 123", "special-tests-123"},
		{"leading and trailing spaces", "  spaced site  ", "spaced-site"},
		{"emoji only", "🚀✨", ""},
		{"overlong not truncated", longName, expectedLongSlug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := create.Slugify(tt.input)
			if actual != tt.expected {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, actual, tt.expected)
			}
		})
	}
}

func TestValidateSlug(t *testing.T) {
	tests := []struct {
		slug    string
		wantErr bool
	}{
		{"valid-slug-123", false},
		{"a", false},
		{"", true},
		{"-starts-with-hyphen", true},
		{"ends-with-hyphen-", true},
		{"has_underscore", true},
		{"has.dot", true},
		{"CON", true},
		{"nul", true},
		{"com1", true},
		{"lpt9", true},
		{strings.Repeat("a", 63), false}, // 63 chars allowed
		{strings.Repeat("a", 64), true},  // 64 chars exceeds 63 max
	}

	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			err := create.ValidateSlug(tt.slug)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSlug(%q) error = %v, wantErr %v", tt.slug, err, tt.wantErr)
			}
		})
	}
}
