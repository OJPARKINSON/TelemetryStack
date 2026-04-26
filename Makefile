.PHONY: restart logs build-push build-push-dashboard build-push-telemetry

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

build-push-dashboard:
	@echo "🔨 Building dashboard image..."
	@docker build -t ghcr.io/ojparkinson/iracing-display:latest ./dashboard
	@echo "📤 Pushing ghcr.io/ojparkinson/iracing-display:latest..."
	@docker push ghcr.io/ojparkinson/iracing-display:latest
	@echo "✅ Dashboard image pushed."

build-push-telemetry:
	@echo "🔨 Building telemetry-service image..."
	@docker build -t ghcr.io/ojparkinson/iracing-telemetryservice:latest ./telemetryService/golang
	@echo "📤 Pushing ghcr.io/ojparkinson/iracing-telemetryservice:latest..."
	@docker push ghcr.io/ojparkinson/iracing-telemetryservice:latest
	@echo "✅ Telemetry service image pushed."

build-push: build-push-dashboard build-push-telemetry
	@echo "🎉 All images built and pushed."
