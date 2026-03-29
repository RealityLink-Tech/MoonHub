package fileutil

import (
	"testing"
)

func TestValidateSafePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"simple", "file.txt", false},
		{"absolute", "/tmp/file.txt", false},
		{"literal_dots_in_name", "foo/..bar/baz", true}, // strings.Contains catches ".." substring (acceptable for defense-in-depth)
		{"resolvable_dots", "foo/../bar", false},             // cleans to "bar"
		{"unresolvable_dots", "a/b/../../../c", true},        // cleans to "../c"
		{"dots_only", "..", true},                            // cleans to ".."
		{"hidden_no_traversal", ".hidden/file", false},
		{"clean_safe", "/var/log/app.log", false},
		{"traversal_resolved", "/safe/../../etc/passwd", false}, // cleans to "/etc/passwd"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSafePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSafePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}
