package phpversion

import "testing"

func TestConstraintSatisfies(t *testing.T) {
	tests := []struct {
		constraint string
		version    string
		want       bool
	}{
		{">=8.2", "8.2.0", true},
		{">=8.2", "8.1.9", false},
		{">=8.2", "8.5.9", true},
		{">8.2", "8.2.0", false},
		{">8.2", "8.2.1", true},
		{"<=8.2", "8.2.0", true},
		{"<=8.2", "8.1.9", true},
		{"<=8.2", "8.2.5", false},
		{"<8.2", "8.1.9", true},
		{"<8.2", "8.2.0", false},
		{"^8.2", "8.2.0", true},
		{"^8.2", "8.9.0", true},
		{"^8.2", "9.0.0", false},
		{"^8.2", "8.1.9", false},
		{"~8.2", "8.9.0", true},
		{"~8.2", "9.0.0", false},
		{"~8.2.1", "8.2.9", true},
		{"~8.2.1", "8.3.0", false},
		{"~8.2.1", "8.2.0", false},
		{"8.2", "8.2.9", true},
		{"8.2", "8.3.0", false},
		{"8.2.*", "8.2.5", true},
		{"8.2.*", "8.3.0", false},
		{"^7.4 || ^8.2", "7.4.5", true},
		{"^7.4 || ^8.2", "8.2.5", true},
		{"^7.4 || ^8.2", "8.0.0", false},
		{"^7.4 || ^8.2", "9.0.0", false},
		{">=8.1,<8.3", "8.2.0", true},
		{">=8.1,<8.3", "8.3.0", false},
	}

	for _, tt := range tests {
		c, err := ParseConstraint(tt.constraint)
		if err != nil {
			t.Fatalf("ParseConstraint(%q) error = %v", tt.constraint, err)
		}
		v, err := ParseVersion(tt.version)
		if err != nil {
			t.Fatalf("ParseVersion(%q) error = %v", tt.version, err)
		}
		if got := c.Satisfies(v); got != tt.want {
			t.Errorf("ParseConstraint(%q).Satisfies(%q) = %v, want %v", tt.constraint, tt.version, got, tt.want)
		}
	}
}

func TestParseConstraintRejectsEmpty(t *testing.T) {
	if _, err := ParseConstraint(""); err == nil {
		t.Error("ParseConstraint(\"\") error = nil, want an error")
	}
	if _, err := ParseConstraint("   "); err == nil {
		t.Error(`ParseConstraint("   ") error = nil, want an error`)
	}
}

func TestParseConstraintRejectsInvalidClause(t *testing.T) {
	if _, err := ParseConstraint(">=not-a-version"); err == nil {
		t.Error("ParseConstraint() error = nil, want an error for an invalid clause")
	}
}
