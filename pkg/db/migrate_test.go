package db

import "testing"

func TestParseVersion(t *testing.T) {
	tests := []struct {
		filename string
		want     int64
		wantErr  bool
	}{
		{"000001_initial.up.sql", 1, false},
		{"000012_add_hosts.up.sql", 12, false},
		{"100_short.up.sql", 100, false},
		{"bad.up.sql", 0, true},
	}
	for _, tt := range tests {
		got, err := parseVersion(tt.filename)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseVersion(%q) error = %v, wantErr %v", tt.filename, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("parseVersion(%q) = %d, want %d", tt.filename, got, tt.want)
		}
	}
}
