.PHONY: web api cli prompts prompts-sql

web:
	go run ./cmd/web

api:
	go run ./cmd/api

cli:
	go run ./cmd/cli

# Validates the Jev prompt examples (no database): it reads scripts/prompts/exemplos.json,
# runs each row through the exact production path and exits non-zero on any miss.
# FILE=... overrides the examples file; FLOW=... limits to one flow.
prompts:
	go run ./cmd/prompts -file $(if $(FILE),$(FILE),scripts/prompts/exemplos.json) $(if $(FLOW),-flow $(FLOW),)

# Regenerates scripts/sql/seed_prompts.sql from the JSON examples, so the web
# harness and the file runner can never drift.
prompts-sql:
	go run ./cmd/prompts -file $(if $(FILE),$(FILE),scripts/prompts/exemplos.json) -sql scripts/sql/seed_prompts.sql