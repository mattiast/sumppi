package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type SeriesConfig struct {
	Series []Series `toml:"series"`
}

type Series struct {
	GUID   string `toml:"guid"`
	S3Path string `toml:"s3_path"`
}

type APIResponse struct {
	Data SeriesData `json:"data"`
}

type SeriesData struct {
	GUID            string    `json:"guid"`
	LastModified    string    `json:"last_modified"`
	RSSFeedURL      string    `json:"rss_feed_url"`
	Title           string    `json:"title"`
	Author          string    `json:"author"`
	Description     string    `json:"description"`
	HTMLDescription *string   `json:"html_description"`
	Link            string    `json:"link"`
	PublicationDate string    `json:"publication_date"`
	Copyright       string    `json:"copyright"`
	Publisher       string    `json:"publisher"`
	Tags            []string  `json:"tags"`
	Categories      []string  `json:"categories"`
	Episodes        []Episode `json:"episodes"`
	Rankings        Rankings  `json:"rankings"`
	CoverURL        string    `json:"cover_url"`
}

type Episode struct {
	SourceType          string               `json:"source_type"`
	GUID                string               `json:"guid"`
	SeriesTitle         string               `json:"series_title"`
	SeriesGUID          string               `json:"series_guid"`
	Author              string               `json:"author"`
	PhotoAuthor         string               `json:"photo_author"`
	OriginalArticleURL  string               `json:"original_article_url"`
	Title               string               `json:"title"`
	Description         string               `json:"description"`
	HTMLDescription     *string              `json:"html_description"`
	PublicationDate     string               `json:"publication_date"`
	RSSGUID             string               `json:"rss_guid"`
	AudioURL            string               `json:"audio_url"`
	AudioDuration       int                  `json:"audio_duration"`
	AudioLength         int                  `json:"audio_length"`
	AudioSample         AudioSample          `json:"audio_sample"`
	AudioPkgs           map[string]string    `json:"audio_pkgs"`
	LastModified        string               `json:"last_modified"`
	AudioSlices         []AudioSlice         `json:"audio_slices"`
	SeriesTags          []string             `json:"series_tags"`
	Tags                []string             `json:"tags"`
	AvailabilityPeriods []AvailabilityPeriod `json:"availability_periods"`
	Rankings            Rankings             `json:"rankings"`
	AnalyticsData       *string              `json:"analytics_data"`
	AdTags              *string              `json:"ad_tags"`
	CoverURL            string               `json:"cover_url"`
	SquareCoverURL      *string              `json:"square_cover_url"`
	SquarePhotoAuthor   *string              `json:"square_photo_author"`
	H                   *string              `json:"h"`
	Kind                string               `json:"kind"`
}

type AudioSample struct {
	AudioURL      string `json:"audio_url"`
	AudioDuration int    `json:"audio_duration"`
	AudioLength   int    `json:"audio_length"`
}

type AudioSlice struct {
	URL   string `json:"url"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

type AvailabilityPeriod struct {
	Product   *string `json:"product"`
	Type      string  `json:"type"`
	StartDate string  `json:"start_date"`
	EndDate   string  `json:"end_date"`
}

type Rankings struct {
	Daily   int `json:"daily"`
	Weekly  int `json:"weekly"`
	Monthly int `json:"monthly"`
}

func fetchSeriesData(guid string) (*SeriesData, error) {
	url := fmt.Sprintf("https://appdata.richie.fi/books/feeds/v3/Nelonen/podcast_series/%s.json", guid)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch series data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status code %d", resp.StatusCode)
	}

	var apiResponse APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	return &apiResponse.Data, nil
}

func getLatestEpisodeDate(episodes []Episode) (string, error) {
	if len(episodes) == 0 {
		return "", fmt.Errorf("no episodes found")
	}

	latestDate := episodes[0].PublicationDate
	for _, episode := range episodes[1:] {
		if episode.PublicationDate > latestDate {
			latestDate = episode.PublicationDate
		}
	}

	// Parse and format the date nicely
	parsedTime, err := time.Parse(time.RFC3339, latestDate)
	if err != nil {
		// If parsing fails, return the original date
		return latestDate, nil
	}

	return parsedTime.Format("Jan 2, 2006"), nil
}
