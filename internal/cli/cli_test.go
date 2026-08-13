package cli

import (
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{
			name:       "version flag prints version and commit",
			args:       []string{"--version"},
			wantCode:   exitOK,
			wantStdout: "spin 1.2.3 (abc123)\n",
		},
		{
			name:       "version word prints version and commit",
			args:       []string{"version"},
			wantCode:   exitOK,
			wantStdout: "spin 1.2.3 (abc123)\n",
		},
		{
			name:     "help flag prints usage to stdout",
			args:     []string{"--help"},
			wantCode: exitOK,
		},
		{
			name:     "no arguments prints usage to stdout and exits with a usage error",
			args:     []string{},
			wantCode: exitUsage,
		},
		{
			name:     "unknown command prints usage to stderr and exits with a usage error",
			args:     []string{"does-not-exist"},
			wantCode: exitUsage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr strings.Builder

			code := Run(tt.args, &stdout, &stderr, "1.2.3", "abc123")

			if code != tt.wantCode {
				t.Errorf("Run(%v) code = %d, want %d", tt.args, code, tt.wantCode)
			}
			if tt.wantStdout != "" && stdout.String() != tt.wantStdout {
				t.Errorf("Run(%v) stdout = %q, want %q", tt.args, stdout.String(), tt.wantStdout)
			}
			if tt.wantStderr != "" && !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("Run(%v) stderr = %q, want it to contain %q", tt.args, stderr.String(), tt.wantStderr)
			}
		})
	}
}

func TestRun_unknownCommandMentionsItByName(t *testing.T) {
	var stdout, stderr strings.Builder

	Run([]string{"bogus"}, &stdout, &stderr, "1.2.3", "abc123")

	if !strings.Contains(stderr.String(), "bogus") {
		t.Errorf("stderr = %q, want it to mention the unknown command name", stderr.String())
	}
}
