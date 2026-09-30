package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testNow() time.Time {
	return time.Date(2026, 3, 15, 18, 45, 0, 0, time.UTC)
}

func TestDateParserParseDate(t *testing.T) {
	parser := NewDateParser()
	now := testNow()

	tests := []struct {
		name    string
		message string
		want    time.Time
		wantOK  bool
	}{
		{"full date", "pagar a conta 10/05/2026", time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC), true},
		{"full date single digits", "consulta 1/2/2027", time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC), true},
		{"day and month uses current year", "10/05", time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC), true},
		{"dia N uses current month", "pagar a conta dia 10", time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC), true},
		{"dia N no dia", "consulta no dia 7", time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC), true},
		{"hoje", "quais são minhas notas de hoje?", time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC), true},
		{"amanha", "me lembra amanhã", time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC), true},
		{"ontem", "o que eu anotei ontem?", time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC), true},
		{"invalid day for month", "31/02/2026", time.Time{}, false},
		{"invalid day number", "dia 45", time.Time{}, false},
		{"no date", "anota que preciso comprar pão", time.Time{}, false},
		{"empty", "", time.Time{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parser.ParseDate(tt.message, now)
			assert.Equal(t, tt.wantOK, ok)
			assert.True(t, got.Equal(tt.want), "date = %v, want %v", got, tt.want)
		})
	}
}

func TestDateParserParseDateAlwaysUTC(t *testing.T) {
	parser := NewDateParser()

	// A non-UTC "now" must still yield UTC midnight of the same calendar day.
	got, ok := parser.ParseDate("hoje", time.Date(2026, 3, 15, 23, 30, 0, 0, time.FixedZone("BRT", -3*60*60)))
	require.True(t, ok)
	assert.Equal(t, time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC), got)
	assert.Equal(t, time.UTC, got.Location())
}

func TestDateParserParseTime(t *testing.T) {
	parser := NewDateParser()

	tests := []struct {
		name    string
		message string
		want    string
		wantOK  bool
	}{
		{"hour only", "consulta 10/05/2026 às 14h", "14:00", true},
		{"hour and minutes", "consulta 10/05/2026 às 14h30", "14:30", true},
		{"colon", "consulta 10/05/2026 às 14:05", "14:05", true},
		{"no space", "às 9h", "09:00", true},
		{"invalid hour", "às 25h", "", false},
		{"invalid minutes", "às 14h75", "", false},
		{"no time", "pagar a conta dia 10", "", false},
		{"empty", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parser.ParseTime(tt.message)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}