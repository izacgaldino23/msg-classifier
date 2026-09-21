package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Env struct {
	TypesafeApiUrl string
	TypesafeModel  string
	TypesafeToken  string
}

var env *Env

func GetEnv() *Env {
	if env == nil {
		_ = godotenv.Load("local.env")

		env = &Env{
			TypesafeApiUrl: os.Getenv("TYPESAFE_API_URL"),
			TypesafeModel:  os.Getenv("TYPESAFE_MODEL"),
			TypesafeToken:  os.Getenv("TYPESAFE_TOKEN"),
		}
	}

	return env
}
