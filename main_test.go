package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigMaxAgeDays(t *testing.T) {
	tests := []struct {
		name    string
		setting string
		want    int
		wantErr bool
	}{
		{name: "omitted means unlimited"},
		{name: "zero means unlimited", setting: "max_age_days = 0"},
		{name: "positive limit", setting: "max_age_days = 60", want: 60},
		{name: "negative rejected", setting: "max_age_days = -1", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "series.toml")
			// Leave the first feed unlimited to verify that a limit affects only its series.
			contents := "[[series]]\nguid = \"unlimited\"\ns3_path = \"s3://bucket/unlimited.rss\"\n\n" +
				"[[series]]\nguid = \"selected\"\ns3_path = \"s3://bucket/selected.rss\"\n" + tt.setting + "\n"
			if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("SUMPPI_CONFIG", path)
			config, err := loadConfig()
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "series 2 (selected): max_age_days must be non-negative") {
					t.Fatalf("expected actionable max_age_days error, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(config.Series) != 2 {
				t.Fatalf("got %d series, want 2", len(config.Series))
			}
			if config.Series[0].MaxAgeDays != 0 || config.Series[1].MaxAgeDays != tt.want {
				t.Errorf("limits = (%d, %d), want (0, %d)", config.Series[0].MaxAgeDays, config.Series[1].MaxAgeDays, tt.want)
			}
		})
	}
}
