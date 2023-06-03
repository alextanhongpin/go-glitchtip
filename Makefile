include .env
export


run:
	@go run main.go


up:
	@docker-compose up -d


down:
	@docker-compose down


open:
	@open http://localhost:8000
