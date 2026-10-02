package ptbr

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
		{"long date", "paguei 10 de outubro de 2023", time.Date(2023, 10, 10, 0, 0, 0, 0, time.UTC), true},
		{"long date accented month", "paguei 10 de março de 2023", time.Date(2023, 3, 10, 0, 0, 0, 0, time.UTC), true},
		{"month and day uses current year", "15 de novembro", time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC), true},
		{"dash date", "10-10-2023", time.Date(2023, 10, 10, 0, 0, 0, 0, time.UTC), true},
		{"invalid long date", "31 de fevereiro de 2023", time.Time{}, false},
		{"period is not a single day", "compras na semana passada", time.Time{}, false},
		{"no date", "anota que preciso comprar pão", time.Time{}, false},
		{"empty", "", time.Time{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseDate(tt.message, now)
			assert.Equal(t, tt.wantOK, ok)
			assert.True(t, got.Equal(tt.want), "date = %v, want %v", got, tt.want)
		})
	}
}

func TestDateParserParseDateAlwaysUTC(t *testing.T) {
	// A non-UTC "now" must still yield UTC midnight of the same calendar day.
	got, ok := ParseDate("hoje", time.Date(2026, 3, 15, 23, 30, 0, 0, time.FixedZone("BRT", -3*60*60)))
	require.True(t, ok)
	assert.Equal(t, time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC), got)
	assert.Equal(t, time.UTC, got.Location())
}

func TestDateParserParseTime(t *testing.T) {
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
			got, ok := ParseTime(tt.message)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseRange(t *testing.T) {
	now := testNow()

	tests := []struct {
		name      string
		message   string
		wantFrom  time.Time
		wantUntil time.Time
		wantOK    bool
	}{
		{"exact date is a single day", "10/10/2023", date(2023, 10, 10), date(2023, 10, 11), true},
		{"hoje", "quanto gastei hoje?", date(2026, 3, 15), date(2026, 3, 16), true},
		{"ontem", "quando gastei ontem?", date(2026, 3, 14), date(2026, 3, 15), true},
		{"semana passada is a rolling week", "compras na semana passada", date(2026, 3, 8), date(2026, 3, 15), true},
		{"essa semana", "quanto gastei essa semana?", date(2026, 3, 9), date(2026, 3, 16), true},
		{"semana que vem", "gastos da semana que vem", date(2026, 3, 15), date(2026, 3, 23), true},
		{"mes passado", "quanto gastei no mês passado?", date(2026, 2, 1), date(2026, 3, 1), true},
		{"esse mes", "quanto gastei esse mês?", date(2026, 3, 1), date(2026, 4, 1), true},
		{"mes que vem", "gastos do mês que vem", date(2026, 4, 1), date(2026, 5, 1), true},
		{"esse ano", "total do ano", date(2026, 1, 1), date(2027, 1, 1), true},
		{"month name", "quanto gastei com mercado em outubro?", date(2026, 10, 1), date(2026, 11, 1), true},
		{"no period", "comprei pão", time.Time{}, time.Time{}, false},
		{"empty", "", time.Time{}, time.Time{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, until, ok := ParseRange(tt.message, now)
			assert.Equal(t, tt.wantOK, ok)
			assert.True(t, from.Equal(tt.wantFrom), "from = %v, want %v", from, tt.wantFrom)
			assert.True(t, until.Equal(tt.wantUntil), "until = %v, want %v", until, tt.wantUntil)
		})
	}
}

func TestParseEventDate(t *testing.T) {
	now := testNow()

	// An exact date is the event day.
	got, ok := ParseEventDate("paguei 50 reais em 10/10/2023", now)
	require.True(t, ok)
	assert.True(t, got.Equal(date(2023, 10, 10)))

	// A period collapses to the last day of the period, since a transaction is one day.
	got, ok = ParseEventDate("comprei arroz semana passada por 50 reais", now)
	require.True(t, ok)
	assert.True(t, got.Equal(date(2026, 3, 14)), "got %v", got)

	// A relative word still wins over nothing.
	got, ok = ParseEventDate("paguei 50 reais hoje", now)
	require.True(t, ok)
	assert.True(t, got.Equal(date(2026, 3, 15)))

	_, ok = ParseEventDate("paguei 50 reais", now)
	assert.False(t, ok, "a message with no period has no event day")
}

// date is a short UTC-midnight helper for the range expectations.
func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}