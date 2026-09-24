.PHONY: help restart logs test test-ingest test-v test-short coverage coverage-html bench bench-mmap bench-parser bench-struct bench-tick bench-all bench-save clean-test

# Default target
.DEFAULT_GOAL := help

help:
	@echo "📚 Available Make Targets:"
	@echo ""
	@echo "  Docker Commands:"
	@echo "    make restart        - Restart Docker services (clean rebuild)"
	@echo "    make logs           - Follow Docker logs"
	@echo ""
	@echo "  Testing Commands:"
	@echo "    make test           - Run ibt submodule tests with race detector"
	@echo "    make test-ingest    - Run this module's own tests (internal/...)"
	@echo "    make test-v         - Run tests (verbose)"
	@echo "    make test-short     - Run short tests only"
	@echo "    make coverage       - Generate test coverage report"
	@echo "    make coverage-html  - Generate and open HTML coverage report"
	@echo ""
	@echo "  Benchmark Commands:"
	@echo "    make bench          - Run all benchmarks (3s each)"
	@echo "    make bench-mmap     - Run mmap benchmarks only"
	@echo "    make bench-parser   - Run parser benchmarks only"
	@echo "    make bench-struct   - Run struct parser benchmarks"
	@echo "    make bench-tick     - Run tick benchmarks"
	@echo "    make bench-all      - Run comprehensive benchmarks (5s each)"
	@echo "    make bench-save     - Run benchmarks and save results with timestamp"
	@echo ""
	@echo "  Cleanup Commands:"
	@echo "    make clean-test     - Remove test artifacts and profiles"
	@echo ""

restart:
	@echo "🚀 Restarting Docker services..."
	@docker-compose down -v
	@docker-compose build --no-cache
	@docker-compose up -d
	@echo "✅ Done! Check logs with: make logs"

logs:
	@docker-compose logs -f go_app

# Testing targets
test:
	@echo "🧪 Running ibt submodule tests..."
	@cd ibt && go test -v -race -timeout 30s ./...

# The targets above test the ibt submodule; this one tests this module.
# The corpus codec measurement is opt-in via WIRE_CODECS (see internal/processing/codec_test.go).
test-ingest:
	@echo "🧪 Running ingest tests..."
	@go test ./internal/... -race

test-v:
	@echo "🧪 Running all tests (verbose)..."
	@cd ibt && go test -v -race -timeout 30s ./...

test-short:
	@echo "🧪 Running short tests..."
	@cd ibt && go test -short -race -timeout 10s ./...

# Coverage targets
coverage:
	@echo "📊 Generating coverage report..."
	@cd ibt && go test -coverprofile=coverage.out ./...
	@cd ibt && go tool cover -func=coverage.out
	@echo ""
	@echo "📄 Coverage profile saved to ibt/coverage.out"
	@echo "💡 Run 'make coverage-html' to view in browser"

coverage-html:
	@echo "📊 Generating HTML coverage report..."
	@cd ibt && go test -coverprofile=coverage.out ./...
	@cd ibt && go tool cover -html=coverage.out
	@echo "✅ Coverage report opened in browser"

# Benchmark targets
bench:
	@echo "⚡ Running all benchmarks..."
	@cd ibt && go test -bench=. -run=^$$ -benchmem -benchtime=3s

bench-mmap:
	@echo "⚡ Running mmap benchmarks..."
	@cd ibt && go test -bench=BenchmarkMmap -run=^$$ -benchmem -benchtime=3s

bench-parser:
	@echo "⚡ Running parser benchmarks..."
	@cd ibt && go test -bench=BenchmarkParser -run=^$$ -benchmem -benchtime=3s

bench-struct:
	@echo "⚡ Running struct benchmarks..."
	@cd ibt && go test -bench=BenchmarkDirectStruct -run=^$$ -benchmem -benchtime=3s

bench-tick:
	@echo "⚡ Running tick benchmarks..."
	@cd ibt && go test -bench=BenchmarkTick -run=^$$ -benchmem -benchtime=3s

bench-all:
	@echo "⚡ Running comprehensive benchmarks (5s each)..."
	@cd ibt && go test -bench=. -run=^$$ -benchmem -benchtime=5s -timeout 30m

# Benchmark comparison (save results)
bench-save:
	@echo "💾 Running benchmarks and saving results..."
	@mkdir -p benchmarks
	@cd ibt && go test -bench=. -run=^$$ -benchmem -benchtime=3s | tee ../benchmarks/bench_$(shell date +%Y%m%d_%H%M%S).txt
	@echo "✅ Results saved to benchmarks/"

# Clean test artifacts
clean-test:
	@echo "🧹 Cleaning test artifacts..."
	@rm -rf ibt/*.test
	@rm -rf ibt/*.prof
	@rm -rf ibt/coverage.out
	@echo "✅ Clean complete"