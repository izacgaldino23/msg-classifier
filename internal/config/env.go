package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Env struct {
	TypesafeApiUrl string
	TypesafeModel  string
	TypesafeToken  string
	DBPath         string
	ApiPort        string
}

var env *Env

func GetEnv() *Env {
	if env == nil {
		_ = godotenv.Load("local.env")

		env = &Env{
			TypesafeApiUrl: os.Getenv("TYPESAFE_API_URL"),
			TypesafeModel:  os.Getenv("TYPESAFE_MODEL"),
			TypesafeToken:  os.Getenv("TS_API_KEY"),
			DBPath:         envOr("DB_PATH", "contacts.db"),
			ApiPort:        envOr("API_PORT", "8081"),
		}
	}

	return env
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
