install:
	@go mod tidy
	@go get github.com/google/uuid
	@go get github.com/mattn/go-sqlite3@v1.14.16
	@go install github.com/pressly/goose/v3/cmd/goose@v3.20.0
	@brew install sqlite3
	@brew install sqlite-utils

dbmigrate_create:
	@goose sqlite3 ./test.db create <MIGRATION_NAME> sql

dbmigrate_up:
	@goose sqlite3 ./test.db up

dbmigrate_down:
	@goose sqlite3 ./test.db down

job_manager_run:
	@go build -o job_manager .
	@./job_manager