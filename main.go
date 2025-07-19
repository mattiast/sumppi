package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	series   []Series
	cursor   int
	selected map[int]struct{}
	loading  bool
	status   string
	s3Client *S3Client
}

func initialModel() model {
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	s3Client, err := NewS3Client(context.Background())
	if err != nil {
		log.Printf("Warning: Failed to initialize S3 client: %v", err)
	}

	return model{
		series:   config.Series,
		selected: make(map[int]struct{}),
		s3Client: s3Client,
	}
}

func loadConfig() (*SeriesConfig, error) {
	configPath := "series.toml"
	if envPath := os.Getenv("SUMPPI_CONFIG"); envPath != "" {
		configPath = envPath
	}

	var config SeriesConfig
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	return &config, nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.series)-1 {
				m.cursor++
			}
		case "enter", " ":
			if !m.loading {
				m.loading = true
				return m, m.generateFeed()
			}
		case "u":
			if !m.loading && m.s3Client != nil {
				m.loading = true
				return m, m.generateAndUploadFeed()
			}
		case "c":
			if !m.loading {
				return m, m.copyURLToClipboard()
			}
		case "d":
			if !m.loading {
				m.loading = true
				return m, m.showLatestEpisodeDate()
			}
		}
	case feedResult:
		m.loading = false
		m.status = string(msg)
	}

	return m, nil
}

type feedResult string

func (m model) generateFeed() tea.Cmd {
	return func() tea.Msg {
		series := m.series[m.cursor]

		seriesData, err := fetchSeriesData(series.GUID)
		if err != nil {
			return feedResult(fmt.Sprintf("Error fetching series data: %v", err))
		}

		rssXML, err := generateRSSFeed(seriesData)
		if err != nil {
			return feedResult(fmt.Sprintf("Error generating RSS feed: %v", err))
		}

		// Extract filename from S3 path
		filename := filepath.Base(series.S3Path)
		if !strings.HasSuffix(filename, ".rss") {
			filename = fmt.Sprintf("%s.rss", series.GUID)
		}

		err = os.WriteFile(filename, []byte(rssXML), 0644)
		if err != nil {
			return feedResult(fmt.Sprintf("Error writing RSS file: %v", err))
		}

		return feedResult(fmt.Sprintf("RSS feed written to %s (%s by %s, %d episodes)", filename, seriesData.Title, seriesData.Author, len(seriesData.Episodes)))
	}
}

func (m model) generateAndUploadFeed() tea.Cmd {
	return func() tea.Msg {
		series := m.series[m.cursor]

		seriesData, err := fetchSeriesData(series.GUID)
		if err != nil {
			return feedResult(fmt.Sprintf("Error fetching series data: %v", err))
		}

		rssXML, err := generateRSSFeed(seriesData)
		if err != nil {
			return feedResult(fmt.Sprintf("Error generating RSS feed: %v", err))
		}

		// Upload directly to S3 from memory
		err = m.s3Client.UploadRSSContent(context.Background(), rssXML, series.S3Path)
		if err != nil {
			return feedResult(fmt.Sprintf("Error uploading to S3: %v", err))
		}

		return feedResult(fmt.Sprintf("RSS feed uploaded to %s (%s by %s, %d episodes)", series.S3Path, seriesData.Title, seriesData.Author, len(seriesData.Episodes)))
	}
}

func (m model) copyURLToClipboard() tea.Cmd {
	return func() tea.Msg {
		series := m.series[m.cursor]

		url, err := generateS3URL(series.S3Path)
		if err != nil {
			return feedResult(fmt.Sprintf("Error generating URL: %v", err))
		}

		err = clipboard.WriteAll(url)
		if err != nil {
			return feedResult(fmt.Sprintf("Error copying to clipboard: %v", err))
		}

		return feedResult(fmt.Sprintf("URL copied to clipboard: %s", url))
	}
}

func (m model) showLatestEpisodeDate() tea.Cmd {
	return func() tea.Msg {
		series := m.series[m.cursor]

		seriesData, err := fetchSeriesData(series.GUID)
		if err != nil {
			return feedResult(fmt.Sprintf("Error fetching series data: %v", err))
		}

		latestDate, err := getLatestEpisodeDate(seriesData.Episodes)
		if err != nil {
			return feedResult(fmt.Sprintf("Error finding latest episode: %v", err))
		}

		return feedResult(fmt.Sprintf("Latest episode date: %s", latestDate))
	}
}

func (m model) View() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	normalStyle := lipgloss.NewStyle()
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	s := headerStyle.Render("RSS Feed Generator") + "\n\n"
	s += "Select a series to generate RSS feed:\n\n"

	for i, series := range m.series {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		line := fmt.Sprintf("%s %s", cursor, extractFilename(series.S3Path))
		if m.cursor == i {
			line = selectedStyle.Render(line)
		} else {
			line = normalStyle.Render(line)
		}
		s += line + "\n"
	}

	s3Status := ""
	if m.s3Client != nil {
		s3Status = " • u: upload to S3"
	}
	s += "\n" + statusStyle.Render(fmt.Sprintf("j/k: navigate • enter/space: generate feed%s • d: show latest episode • c: copy URL • q: quit", s3Status))

	if m.loading {
		s += "\n\n" + statusStyle.Render("Generating feed...")
	} else if m.status != "" {
		s += "\n\n" + statusStyle.Render(m.status)
	}

	return s
}

func extractFilename(s3Path string) string {
	filename := filepath.Base(s3Path)
	if filename == "." || filename == "/" {
		return s3Path
	}
	return filename
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}
