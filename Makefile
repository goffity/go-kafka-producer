.PHONY: run

run:
	docker compose up --remove-orphans --build
down:
	docker compose down -v
