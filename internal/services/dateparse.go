package services

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DateParser extracts PT-BR dates and times from a message. now is a parameter
// (not time.Now()) so the relative forms are deterministic in tests.
type DateParser struct{}

func NewDateParser() *DateParser {
	return &DateParser{}
}

var (
	fullDatePattern   = regexp.MustCompile(`\b(\d{1,2})/(\d{1,2})/(\d{4})\b`)
	shortDatePattern  = regexp.MustCompile(`\b(\d{1,2})/(\d{1,2})\b`)
	dayOfMonthPattern = regexp.MustCompile(`\bdia\s+(\d{1,2})\b`)
	colonTimePattern  = regexp.MustCompile(`\b(\d{1,2}):(\d{2})\b`)
	hourTimePattern   = regexp.MustCompile(`\b(\d{1,2})\s*h(?:(\d{2}))?\b`)
)

// dateLayout is the PT-BR display format used for the search term.
const dateLayout = "02/01/2006"

// ParseDate returns the date mentioned in the message, or false when there is
// none. Supported: dd/mm/aaaa, dd/mm (current year), "dia N" (current month),
// "hoje", "amanhã" and "ontem". Every result is UTC midnight so the stored
// value and the FindByDate filter compare identically in SQLite.
func (p *DateParser) ParseDate(message string, now time.Time) (time.Time, bool) {
	if m := fullDatePattern.FindStringSubmatch(message); m != nil {
		return buildDate(toInt(m[1]), toInt(m[2]), toInt(m[3]))
	}
	if m := shortDatePattern.FindStringSubmatch(message); m != nil {
		return buildDate(toInt(m[1]), toInt(m[2]), now.Year())
	}

	normalized := normalizeName(message)
	switch {
	case strings.Contains(normalized, "hoje"):
		return startOfDay(now), true
	case strings.Contains(normalized, "amanha"):
		return startOfDay(now.AddDate(0, 0, 1)), true
	case strings.Contains(normalized, "ontem"):
		return startOfDay(now.AddDate(0, 0, -1)), true
	}

	if m := dayOfMonthPattern.FindStringSubmatch(normalized); m != nil {
		return buildDate(toInt(m[1]), int(now.Month()), now.Year())
	}
	return time.Time{}, false
}

// ParseTime returns the HH:MM time mentioned in the message ("14h", "14h30",
// "14:00") or false when there is none. The time is optional for reminders.
func (p *DateParser) ParseTime(message string) (string, bool) {
	if m := colonTimePattern.FindStringSubmatch(message); m != nil {
		if value, ok := formatTime(toInt(m[1]), toInt(m[2])); ok {
			return value, true
		}
	}
	if m := hourTimePattern.FindStringSubmatch(message); m != nil {
		minute := 0
		if m[2] != "" {
			minute = toInt(m[2])
		}
		if value, ok := formatTime(toInt(m[1]), minute); ok {
			return value, true
		}
	}
	return "", false
}

// buildDate validates day/month/year (rejecting 31/02 and friends) and returns
// UTC midnight.
func buildDate(day, month, year int) (time.Time, bool) {
	if month < 1 || month > 12 || day < 1 || year < 1900 || year > 9999 {
		return time.Time{}, false
	}
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if date.Day() != day || int(date.Month()) != month || date.Year() != year {
		return time.Time{}, false
	}
	return date, true
}

// startOfDay truncates a time to the UTC midnight of its calendar day.
func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// formatTime validates the clock and formats it as HH:MM.
func formatTime(hour, minute int) (string, bool) {
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return "", false
	}
	return fmt.Sprintf("%02d:%02d", hour, minute), true
}

// toInt parses the digits captured by the date/time patterns.
func toInt(s string) int {
	value, _ := strconv.Atoi(s)
	return value
}