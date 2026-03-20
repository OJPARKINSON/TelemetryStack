.PHONY: restart logs

restart:
	@echo "🚀 Restarting Docker services..."
	@docker compose --file docker-compose.yml down -v
	@docker compose --file docker-compose.yml build --no-cache
	@docker compose --file docker-compose.yml up --pull=always -d
	@echo "✅ Done! Check logs with: make logs"

restart-dev:
	@echo "🚀 Restarting Docker services..."
	@docker compose --file docker-compose.dev.yml down -v
	@docker compose --file docker-compose.dev.yml build --no-cache
	@docker compose --file docker-compose.dev.yml up -d
	@echo "✅ Done! Check logs with: make logs"

restart-lite:
	@echo "🚀 Restarting Dashboard..."
	@docker compose --file docker-compose.dev.yml build --no-cache 
	@docker compose --file docker-compose.dev.yml up -d
	@echo "✅ Done! Check logs with: make logs"

restart-p:
	@echo "🚀 Restarting Docker services..."
	@podman compose --file docker-compose.yml down -v
	@podman compose --file docker-compose.yml build --no-cache
	@podman compose --file docker-compose.yml up -d
	@echo "✅ Done! Check logs with: make logs"

restart-dev-p:
	@echo "🚀 Restarting Docker services..."
	@podman compose --file docker-compose.dev.yml down -v
	@podman compose --file docker-compose.dev.yml build --no-cache
	@podman compose --file docker-compose.dev.yml up -d
	@echo "✅ Done! Check logs with: make logs"

restart-lite-p:
	@echo "🚀 Restarting Dashboard..."
	@podman compose --file docker-compose.dev.yml build --no-cache 
	@podman compose --file docker-compose.dev.yml up -d
	@echo "✅ Done! Check logs with: make logs"