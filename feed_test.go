package main

import (
	"encoding/xml"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestGenerateRSSFeedFiltering(t *testing.T) {
	// Cross a daylight saving change to verify that days remain exactly 24 hours.
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.November, 15, 12, 0, 0, 0, loc)
	cutoff := now.Add(-60 * 24 * time.Hour)
	episode := func(guid string, date time.Time) Episode {
		return Episode{
			GUID:            guid,
			Title:           guid,
			PublicationDate: date.Format(time.RFC3339),
			AudioURL:        "https://example.com/" + guid + ".mp3",
		}
	}
	old := episode("old", cutoff.Add(-time.Second))
	// A recent availability date must not override an old publication date.
	old.AvailabilityPeriods = []AvailabilityPeriod{{StartDate: now.Format(time.RFC3339)}}
	data := &SeriesData{
		Title:       "Test podcast",
		Description: "Test description",
		Author:      "Test author",
		CoverURL:    "https://example.com/cover.jpg",
		Episodes: []Episode{
			old,
			episode("recent", now.Add(-24*time.Hour)),
			{GUID: "invalid", PublicationDate: "not-a-date"},
			episode("cutoff", cutoff.In(time.FixedZone("EET", 2*60*60))),
			episode("upcoming", now.Add(24*time.Hour)),
			{GUID: "missing"},
			episode("future-boundary", now.Add(7*24*time.Hour)),
			episode("far-future", now.Add(7*24*time.Hour+time.Second)),
		},
	}

	tests := []struct {
		name        string
		maxAgeDays  int
		allowFuture bool
		wantGUIDs   []string
	}{
		{
			name:       "60 day limit",
			maxAgeDays: 60,
			wantGUIDs:  []string{"recent", "cutoff", "upcoming", "future-boundary"},
		},
		{
			name:      "unlimited history still skips invalid and far future dates",
			wantGUIDs: []string{"old", "recent", "cutoff", "upcoming", "future-boundary"},
		},
		{
			name:        "allow future retains age limit",
			maxAgeDays:  60,
			allowFuture: true,
			wantGUIDs:   []string{"recent", "cutoff", "upcoming", "future-boundary", "far-future"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rssXML, count, err := generateRSSFeedAt(data, tt.allowFuture, Settings{Timezone: "America/New_York"}, tt.maxAgeDays, now)
			if err != nil {
				t.Fatal(err)
			}
			var feed RSSFeed
			if err := xml.Unmarshal([]byte(rssXML), &feed); err != nil {
				t.Fatalf("invalid RSS XML: %v", err)
			}
			var gotGUIDs []string
			for _, item := range feed.Channel.Items {
				gotGUIDs = append(gotGUIDs, item.GUID.Value)
				if _, err := time.Parse(time.RFC1123Z, item.PubDate); err != nil {
					t.Errorf("invalid RSS publication date for %s: %v", item.GUID.Value, err)
				}
			}
			if !reflect.DeepEqual(gotGUIDs, tt.wantGUIDs) {
				t.Errorf("episode GUIDs = %v, want %v", gotGUIDs, tt.wantGUIDs)
			}
			if count != len(tt.wantGUIDs) || count != len(feed.Channel.Items) {
				t.Errorf("reported count = %d, output count = %d, want %d", count, len(feed.Channel.Items), len(tt.wantGUIDs))
			}
			if feed.Channel.Title != data.Title || feed.Channel.Description != data.Description {
				t.Error("channel metadata was not preserved")
			}
		})
	}
}

func TestGenerateRSSFeedEmpty(t *testing.T) {
	now := time.Date(2026, time.October, 15, 12, 0, 0, 0, time.UTC)
	for _, episodes := range [][]Episode{
		nil,
		{
			{PublicationDate: now.Add(-61 * 24 * time.Hour).Format(time.RFC3339)},
			{PublicationDate: "invalid"},
			{PublicationDate: now.Add(8 * 24 * time.Hour).Format(time.RFC3339)},
		},
	} {
		data := &SeriesData{Title: "Empty podcast", Description: "Still here", Episodes: episodes}
		rssXML, count, err := generateRSSFeedAt(data, false, Settings{}, 60, now)
		if err != nil {
			t.Fatal(err)
		}
		var feed RSSFeed
		if err := xml.Unmarshal([]byte(rssXML), &feed); err != nil {
			t.Fatalf("invalid empty RSS XML: %v", err)
		}
		if count != 0 || len(feed.Channel.Items) != 0 {
			t.Errorf("expected no episodes, reported %d, output %d", count, len(feed.Channel.Items))
		}
		if feed.Channel.Title != data.Title || feed.Channel.Description != data.Description {
			t.Error("empty feed lost its channel metadata")
		}
		if !strings.HasPrefix(rssXML, xml.Header) || feed.Version != "2.0" {
			t.Error("empty feed lost its RSS declaration")
		}
	}
}
