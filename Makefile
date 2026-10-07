.PHONY: web api cli

web:
	go run ./cmd/web

api:
	go run ./cmd/api

cli:
	go run ./cmd/cli
