package main

import (
	"encoding/xml"
	"fmt"
	"time"
)

type RSSFeed struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Xmlns   string   `xml:"xmlns:itunes,attr"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title        string `xml:"title"`
	Description  string `xml:"description"`
	ITunesAuthor string `xml:"itunes:author"`
	ITunesImage  Image  `xml:"itunes:image"`
	Items        []Item `xml:"item"`
}

type Image struct {
	Href string `xml:"href,attr"`
}

type GUID struct {
	IsPermaLink string `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

type Item struct {
	Title          string    `xml:"title"`
	Description    string    `xml:"description"`
	PubDate        string    `xml:"pubDate"`
	GUID           GUID      `xml:"guid"`
	Enclosure      Enclosure `xml:"enclosure"`
	ITunesDuration string    `xml:"itunes:duration"`
}

type Enclosure struct {
	URL    string `xml:"url,attr"`
	Length string `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

func formatDuration(seconds int) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, secs)
	}
	return fmt.Sprintf("%d:%02d", minutes, secs)
}

func generateRSSFeed(seriesData *SeriesData) (string, error) {

	feed := RSSFeed{
		Version: "2.0",
		Xmlns:   "http://www.itunes.com/dtds/podcast-1.0.dtd",
		Channel: Channel{
			Title:        seriesData.Title,
			Description:  seriesData.Description,
			ITunesAuthor: seriesData.Author,
			ITunesImage:  Image{Href: seriesData.CoverURL},
		},
	}

	for _, episode := range seriesData.Episodes {
		episodePubDate, err := time.Parse(time.RFC3339, episode.PublicationDate)
		if err != nil {
			episodePubDate = time.Now()
		}

		item := Item{
			Title:       episode.Title,
			Description: episode.Description,
			PubDate:     episodePubDate.Format(time.RFC1123Z),
			GUID:        GUID{IsPermaLink: "false", Value: episode.GUID},
			Enclosure: Enclosure{
				URL:    episode.AudioURL,
				Length: fmt.Sprintf("%d", episode.AudioLength),
				Type:   "audio/mpeg",
			},
			ITunesDuration: formatDuration(episode.AudioDuration),
		}

		feed.Channel.Items = append(feed.Channel.Items, item)
	}

	xmlData, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal XML: %w", err)
	}

	return xml.Header + string(xmlData), nil
}
