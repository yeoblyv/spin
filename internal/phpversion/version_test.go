package phpversion

import "testing"

func TestParseVersion(t *testing.T) {
	tests := []struct {
		in   string
		want Version
	}{
		{"8", Version{Major: 8}},
		{"8.2", Version{Major: 8, Minor: 2}},
		{"8.2.10", Version{Major: 8, Minor: 2, Patch: 10}},
		{"8.3.0-dev", Version{Major: 8, Minor: 3, Patch: 0}},
	}
	for _, tt := range tests {
		got, err := ParseVersion(tt.in)
		if err != nil {
			t.Fatalf("ParseVersion(%q) error = %v", tt.in, err)
		}
		if got != tt.want {
			t.Errorf("ParseVersion(%q) = %+v, want %+v", tt.in, got, tt.want)
		}
	}
}

func TestParseVersionRejectsInvalid(t *testing.T) {
	for _, in := range []string{"", "a.b", "1.2.3.4"} {
		if _, err := ParseVersion(in); err == nil {
			t.Errorf("ParseVersion(%q) error = nil, want an error", in)
		}
	}
}

func TestVersionCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"8.2.0", "8.2.0", 0},
		{"8.1.0", "8.2.0", -1},
		{"8.2.0", "8.1.0", 1},
		{"8.2.1", "8.2.0", 1},
		{"8.2", "8.2.0", 0},
	}
	for _, tt := range tests {
		a, _ := ParseVersion(tt.a)
		b, _ := ParseVersion(tt.b)
		if got := a.Compare(b); got != tt.want {
			t.Errorf("%s.Compare(%s) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestVersionString(t *testing.T) {
	v := Version{Major: 8, Minor: 2, Patch: 10}
	if got, want := v.String(), "8.2.10"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
