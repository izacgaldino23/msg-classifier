package config

import (
	"testing"
)

func TestGetEnvDBPathDefault(t *testing.T) {
	env = nil
	t.Setenv("DB_PATH", "")

	if got := GetEnv().DBPath; got != "contacts.db" {
		t.Errorf("DBPath = %q, want %q", got, "contacts.db")
	}
	env = nil
}

func TestGetEnvApiPortDefault(t *testing.T) {
	env = nil
	t.Setenv("API_PORT", "")

	if got := GetEnv().ApiPort; got != "8081" {
		t.Errorf("ApiPort = %q, want %q", got, "8081")
	}
	env = nil
}

func TestGetEnvDBPathFromEnv(t *testing.T) {
	env = nil
	t.Setenv("DB_PATH", "test.db")

	if got := GetEnv().DBPath; got != "test.db" {
		t.Errorf("DBPath = %q, want %q", got, "test.db")
	}
	env = nil
}
