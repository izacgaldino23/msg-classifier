package services

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// now is a parameter (not time.Now()) so the relative date forms are
// deterministic in tests.

var (
	fullDatePattern   = regexp.MustCompile(`\b(\d{1,2})/(\d{1,2})/(\d{4})\b`)
	shortDatePattern  = regexp.MustCompile(`\b(\d{1,2})/(\d{1,2})\b`)
	dashDatePattern   = regexp.MustCompile(`\b(\d{1,2})-(\d{1,2})-(\d{4})\b`)
	dayOfMonthPattern = regexp.MustCompile(`\bdia\s+(\d{1,2})\b`)
	colonTimePattern  = regexp.MustCompile(`\b(\d{1,2}):(\d{2})\b`)
	hourTimePattern   = regexp.MustCompile(`\b(\d{1,2})\s*h(?:(\d{2}))?\b`)
	// Month names are matched against the accent-normalized message, so "março" is
	// already "marco" here. A bare month name ("em outubro") is a month range, not a
	// single day, so it is only resolved by ParseRange.
	monthNamePattern = regexp.MustCompile(`\b(janeiro|fevereiro|marco|abril|maio|junho|julho|agosto|setembro|outubro|novembro|dezembro)\b`)
	longDatePattern  = regexp.MustCompile(`\b(\d{1,2})\s+de\s+` + monthNamePattern.String() + `\s+de\s+(\d{4})\b`)
	monthDayPattern  = regexp.MustCompile(`\b(\d{1,2})\s+de\s+` + monthNamePattern.String() + `\b`)
	// "ano" only as a whole word: a plain Contains would match "banco" and "irmão".
	yearWordPattern = regexp.MustCompile(`\bano\b`)
)

// monthsByName maps the normalized PT-BR month name to its time.Month.
var monthsByName = map[string]time.Month{
	"janeiro": time.January, "fevereiro": time.February, "marco": time.March,
	"abril": time.April, "maio": time.May, "junho": time.June,
	"julho": time.July, "agosto": time.August, "setembro": time.September,
	"outubro": time.October, "novembro": time.November, "dezembro": time.December,
}

// dateLayout is the PT-BR display format used for the search term.
const dateLayout = "02/01/2006"

// ParseDate returns the date mentioned in the message, or false when there is
// none. Supported: dd/mm/aaaa, dd/mm (current year), DD-MM-YYYY,
// "10 de outubro de 2023", "15 de novembro" (current year), "dia N" (current
// month), "hoje", "amanhã" and "ontem". Every result is UTC midnight so the
// stored value and the FindByDate filter compare identically in SQLite.
//
// Period phrases ("semana passada", "esse mês") are not days and are left to
// ParseRange.
func ParseDate(message string, now time.Time) (time.Time, bool) {
	if m := fullDatePattern.FindStringSubmatch(message); m != nil {
		return buildDate(toInt(m[1]), toInt(m[2]), toInt(m[3]))
	}
	if m := shortDatePattern.FindStringSubmatch(message); m != nil {
		return buildDate(toInt(m[1]), toInt(m[2]), now.Year())
	}
	if m := dashDatePattern.FindStringSubmatch(message); m != nil {
		return buildDate(toInt(m[1]), toInt(m[2]), toInt(m[3]))
	}

	normalized := normalizeName(message)
	if m := longDatePattern.FindStringSubmatch(normalized); m != nil {
		return buildDate(toInt(m[1]), int(monthsByName[m[2]]), toInt(m[3]))
	}
	if m := monthDayPattern.FindStringSubmatch(normalized); m != nil {
		return buildDate(toInt(m[1]), int(monthsByName[m[2]]), now.Year())
	}

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

// ParseRange returns the period the message talks about as [from, until), both
// bounds at UTC midnight and "until" the day after the period ends, so a
// repository filter (date >= from AND date < until) covers whole days. An exact
// date collapses to a single day. Weeks are rolling seven-day windows and "essa
// semana" is the last seven days including today — no weekday-start rule to argue
// about in PT-BR. Returns false when the message names no period.
func ParseRange(message string, now time.Time) (from, until time.Time, ok bool) {
	if date, found := ParseDate(message, now); found {
		return date, date.AddDate(0, 0, 1), true
	}

	normalized := normalizeName(message)
	today := startOfDay(now)
	switch {
	case strings.Contains(normalized, "hoje"):
		return today, today.AddDate(0, 0, 1), true
	case strings.Contains(normalized, "ontem"):
		return today.AddDate(0, 0, -1), today, true
	case strings.Contains(normalized, "semana"):
		switch {
		case mentionsAny(normalized, "semana passada", "ultima semana", "semana anterior"):
			return today.AddDate(0, 0, -7), today, true
		case mentionsAny(normalized, "semana que vem", "proxima semana", "semana seguinte"):
			return today, today.AddDate(0, 0, 8), true
		default:
			return today.AddDate(0, 0, -6), today.AddDate(0, 0, 1), true
		}
	case mentionsAny(normalized, "mes passado", "mes anterior", "ultimo mes"),
		mentionsAny(normalized, "mes que vem", "proximo mes", "mes seguinte"),
		mentionsAny(normalized, "esse mes", "este mes", "mes atual"):
		offset := 0
		if mentionsAny(normalized, "mes passado", "mes anterior", "ultimo mes") {
			offset = -1
		} else if mentionsAny(normalized, "mes que vem", "proximo mes", "mes seguinte") {
			offset = 1
		}
		first := monthStart(today.Year(), today.Month(), offset)
		return first, first.AddDate(0, 1, 0), true
	case yearWordPattern.MatchString(normalized):
		first := monthStart(today.Year(), time.January, 0)
		return first, first.AddDate(1, 0, 0), true
	}

	if m := monthNamePattern.FindStringSubmatch(normalized); m != nil {
		first := monthStart(today.Year(), monthsByName[m[1]], 0)
		return first, first.AddDate(0, 1, 0), true
	}
	return time.Time{}, time.Time{}, false
}

// ParseEventDate returns the single day an event happened on, for a flow that
// stores one day per record (a transaction). An exact date wins; a period
// ("comprei semana passada") collapses to the last day of the period — a
// transaction is one day and the message only said the week.
func ParseEventDate(message string, now time.Time) (time.Time, bool) {
	if date, ok := ParseDate(message, now); ok {
		return date, true
	}
	_, until, ok := ParseRange(message, now)
	if !ok {
		return time.Time{}, false
	}
	return until.AddDate(0, 0, -1), true
}

// mentionsAny reports whether the normalized message contains any of the phrases.
func mentionsAny(normalized string, phrases ...string) bool {
	for _, phrase := range phrases {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}
	return false
}

// monthStart returns UTC midnight of the first day of the month, shifted by the
// given number of months.
func monthStart(year int, month time.Month, offset int) time.Time {
	return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).AddDate(0, offset, 0)
}


// ParseTime returns the HH:MM time mentioned in the message ("14h", "14h30",
// "14:00") or false when there is none. The time is optional for reminders.
func ParseTime(message string) (string, bool) {
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