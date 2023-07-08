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


trigger:
	#@curl -XPOST localhost:8080/usecase -H 'Content-Type: application/json' -d '{"name": "john"}'
	#@curl localhost:8080/message
	@curl localhost:8080/error -H 'Content-Type: application/json' -d '{"name": "john"}'
