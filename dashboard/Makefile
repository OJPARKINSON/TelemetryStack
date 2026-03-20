.PHONY: dev prod build-dev build-prod start-dev start-prod restart-dev restart-prod logs-dev logs-prod down network clean help

dev: network build-dev start-dev
	@echo "✅ Development environment is up and running at http://localhost:3000"
	@echo "📝 View logs with: make logs-dev"

build-dev:
	@echo "🔨 Building development Docker image..."
	@docker-compose -f docker-compose.dev.yml build

start-dev:
	@echo "🚀 Starting development Docker services..."
	@docker-compose -f docker-compose.dev.yml up -d

restart-dev:
	@echo "🔄 Restarting development Docker services..."
	@docker-compose -f docker-compose.dev.yml down
	@docker-compose -f docker-compose.dev.yml build
	@docker-compose -f docker-compose.dev.yml up -d
	@echo "✅ Development environment restarted! Check logs with: make logs-dev"

logs-dev:
	@echo "📋 Following development logs..."
	@docker-compose -f docker-compose.dev.yml logs -f telemetry-dashboard

prod: network build-prod start-prod
	@echo "✅ Production environment is up and running at http://localhost:3000"
	@echo "📝 View logs with: make logs-prod"

build-prod:
	@echo "🔨 Building production Docker image..."
	@docker-compose build

start-prod:
	@echo "🚀 Starting production Docker services..."
	@docker-compose up -d

restart-prod:
	@echo "🔄 Restarting production Docker services..."
	@docker-compose down
	@docker-compose build --no-cache
	@docker-compose up -d
	@echo "✅ Production environment restarted! Check logs with: make logs-prod"

logs-prod:
	@echo "📋 Following production logs..."
	@docker-compose logs -f telemetry-dashboard

network:
	@echo "🌐 Ensuring shared network exists..."
	@docker network inspect shared-network >/dev/null 2>&1 || docker network create shared-network

down:
	@echo "🛑 Stopping all Docker services..."
	@docker-compose down || true
	@docker-compose -f docker-compose.dev.yml down || true
	@echo "✅ All services stopped"

clean: down
	@echo "🧹 Cleaning up Docker resources..."
	@docker-compose down -v || true
	@docker-compose -f docker-compose.dev.yml down -v || true
	@echo "🗑️ Removing development and production images..."
	@docker rmi -f telemetry-dashboard telemetry-dashboard-dev 2>/dev/null || true
	@echo "✅ Clean up complete"

restart: restart-prod

logs: logs-prod

help:
	@echo "📚 Available commands:"
	@echo "    make dev           - Build and start development environment"
	@echo "    make prod          - Build and start production environment"
	@echo "    make restart-dev   - Restart development environment"
	@echo "    make restart-prod  - Restart production environment"
	@echo "    make logs-dev      - View development logs"
	@echo "    make logs-prod     - View production logs"
	@echo "    make down          - Stop all services"
	@echo "    make clean         - Stop all services and remove volumes/images"
	@echo "    make help          - Show this help message"

# Default target
.DEFAULT_GOAL := Help