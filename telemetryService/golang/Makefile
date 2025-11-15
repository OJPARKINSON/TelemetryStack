.PHONY: help build run test bench bench-mem bench-cpu bench-compare clean profile-mem profile-cpu view-trace restart logs

# Directories
BENCH_DIR := benchmark_results

# Default target
.DEFAULT_GOAL := help

help:
	@echo "📚 Available Make Targets:"
	@echo ""
	@echo "  Docker Commands:"
	@echo "    make restart        - Restart Docker services (clean rebuild)"
	@echo "    make logs           - Follow Docker logs for telemetry service"
	@echo ""
	@echo "  Build & Run:"
	@echo "    make build          - Build the telemetry service"
	@echo "    make run            - Run the telemetry service locally"
	@echo ""
	@echo "  Testing Commands:"
	@echo "    make test           - Run all tests"
	@echo ""
	@echo "  Benchmark Commands:"
	@echo "    make bench          - Run all benchmarks with summary"
	@echo "    make bench-mem      - Run benchmarks with memory profiling"
	@echo "    make bench-cpu      - Run benchmarks with CPU profiling"
	@echo "    make bench-trace    - Run benchmarks with execution trace"
	@echo "    make bench-compare  - Compare current benchmarks with baseline"
	@echo "    make bench-save     - Save benchmark results with timestamp"
	@echo ""
	@echo "  Analysis Tools:"
	@echo "    make profile-mem    - View memory profile interactively"
	@echo "    make profile-cpu    - View CPU profile interactively"
	@echo "    make view-trace     - View execution trace"
	@echo ""
	@echo "  Cleanup Commands:"
	@echo "    make clean          - Clean benchmark results and profiles"
	@echo ""

# Docker commands
restart:
	@echo "🚀 Restarting telemetry service..."
	@cd ../.. && docker compose down telemetry-service
	@cd ../.. && docker compose build --no-cache telemetry-service
	@cd ../.. && docker compose up -d telemetry-service
	@echo "✅ Done! Check logs with: make logs"

logs:
	@cd ../.. && docker compose logs -f telemetry-service

# Build the service
build:
	@echo "🔨 Building telemetry service..."
	@go build -o bin/telemetry-service ./main.go
	@echo "✅ Build complete!"

# Run the service
run:
	@echo "🚀 Running telemetry service..."
	@go run ./main.go

# Run tests
test:
	@echo "🧪 Running all tests..."
	@go test -v -race -timeout 30s ./...

# Run all benchmarks with detailed output
bench:
	@echo "⚡ Running all benchmarks..."
	@mkdir -p $(BENCH_DIR)
	@echo ""
	@echo "--- Subscriber Benchmarks ---"
	@go test -bench=. -benchmem -benchtime=3s ./internal/queue | tee $(BENCH_DIR)/subscriber_bench.txt
	@echo ""
	@echo "--- Persistance Benchmarks ---"
	@go test -bench=. -benchmem -benchtime=3s ./internal/persistance | tee $(BENCH_DIR)/persistance_bench.txt
	@echo ""
	@echo "✅ Results saved to $(BENCH_DIR)/"

# Run benchmarks with memory profiling
bench-mem:
	@echo "📊 Running Benchmarks with Memory Profiling..."
	@mkdir -p $(BENCH_DIR)
	@echo "Profiling subscriber package..."
	@go test -bench=. -benchmem -memprofile=$(BENCH_DIR)/subscriber_mem.prof -benchtime=3s ./internal/queue > $(BENCH_DIR)/subscriber_bench.txt
	@echo "Profiling persistance package..."
	@go test -bench=. -benchmem -memprofile=$(BENCH_DIR)/persistance_mem.prof -benchtime=3s ./internal/persistance > $(BENCH_DIR)/persistance_bench.txt
	@echo ""
	@echo "=== Memory Profile Summary ==="
	@echo ""
	@echo "--- Top Memory Allocations (Subscriber) ---"
	@go tool pprof -top -alloc_space $(BENCH_DIR)/subscriber_mem.prof 2>/dev/null | head -n 15
	@echo ""
	@echo "--- Top Memory Allocations (Persistance) ---"
	@go tool pprof -top -alloc_space $(BENCH_DIR)/persistance_mem.prof 2>/dev/null | head -n 15
	@echo ""
	@echo "✅ Profiles saved to $(BENCH_DIR)/"
	@echo "💡 View interactively: make profile-mem"

# Run benchmarks with CPU profiling
bench-cpu:
	@echo "⚡ Running Benchmarks with CPU Profiling..."
	@mkdir -p $(BENCH_DIR)
	@echo "Profiling subscriber package..."
	@go test -bench=. -cpuprofile=$(BENCH_DIR)/subscriber_cpu.prof -benchtime=3s ./internal/queue > /dev/null
	@echo "Profiling persistance package..."
	@go test -bench=. -cpuprofile=$(BENCH_DIR)/persistance_cpu.prof -benchtime=3s ./internal/persistance > /dev/null
	@echo ""
	@echo "=== CPU Profile Summary ==="
	@echo ""
	@echo "--- Top CPU Usage (Subscriber) ---"
	@go tool pprof -top $(BENCH_DIR)/subscriber_cpu.prof 2>/dev/null | head -n 15
	@echo ""
	@echo "--- Top CPU Usage (Persistance) ---"
	@go tool pprof -top $(BENCH_DIR)/persistance_cpu.prof 2>/dev/null | head -n 15
	@echo ""
	@echo "✅ Profiles saved to $(BENCH_DIR)/"
	@echo "💡 View interactively: make profile-cpu"

# Run benchmarks with execution trace
bench-trace:
	@echo "🔍 Running Benchmarks with Execution Trace..."
	@mkdir -p $(BENCH_DIR)
	@echo "Tracing subscriber package..."
	@go test -bench=BenchmarkFlushBatches/Large -trace=$(BENCH_DIR)/subscriber_trace.out -benchtime=1s ./internal/queue > /dev/null
	@echo "Tracing persistance package..."
	@go test -bench=BenchmarkBatchSerialization/5000 -trace=$(BENCH_DIR)/persistance_trace.out -benchtime=1s ./internal/persistance > /dev/null
	@echo ""
	@echo "✅ Traces saved to $(BENCH_DIR)/"
	@echo "💡 View with: make view-trace"

# Save benchmark results with timestamp
bench-save:
	@echo "💾 Running benchmarks and saving results..."
	@mkdir -p benchmarks
	@go test -bench=. -benchmem -benchtime=3s ./internal/queue ./internal/persistance | tee benchmarks/bench_$(shell date +%Y%m%d_%H%M%S).txt
	@echo "✅ Results saved to benchmarks/"

# Compare benchmarks with baseline (requires benchstat)
bench-compare:
	@if [ ! -f $(BENCH_DIR)/baseline_subscriber.txt ]; then \
		echo "📊 No baseline found. Creating baseline from current results..."; \
		$(MAKE) bench; \
		cp $(BENCH_DIR)/subscriber_bench.txt $(BENCH_DIR)/baseline_subscriber.txt; \
		cp $(BENCH_DIR)/persistance_bench.txt $(BENCH_DIR)/baseline_persistance.txt; \
		echo "✅ Baseline created. Make code changes and run 'make bench-compare' again."; \
	else \
		echo "📊 Comparing with Baseline..."; \
		$(MAKE) bench; \
		echo ""; \
		echo "--- Subscriber Comparison ---"; \
		benchstat $(BENCH_DIR)/baseline_subscriber.txt $(BENCH_DIR)/subscriber_bench.txt || echo "⚠️  Install benchstat: make install-benchstat"; \
		echo ""; \
		echo "--- Persistance Comparison ---"; \
		benchstat $(BENCH_DIR)/baseline_persistance.txt $(BENCH_DIR)/persistance_bench.txt || echo "⚠️  Install benchstat: make install-benchstat"; \
	fi

# View memory profile interactively
profile-mem:
	@if [ ! -f $(BENCH_DIR)/subscriber_mem.prof ]; then \
		echo "⚠️  No memory profile found. Run 'make bench-mem' first."; \
		exit 1; \
	fi
	@echo "🔍 Opening memory profile (subscriber)..."
	@echo "💡 Commands: top, list <function>, web, pdf, help"
	@go tool pprof $(BENCH_DIR)/subscriber_mem.prof

# View CPU profile interactively
profile-cpu:
	@if [ ! -f $(BENCH_DIR)/subscriber_cpu.prof ]; then \
		echo "⚠️  No CPU profile found. Run 'make bench-cpu' first."; \
		exit 1; \
	fi
	@echo "🔍 Opening CPU profile (subscriber)..."
	@echo "💡 Commands: top, list <function>, web, pdf, help"
	@go tool pprof $(BENCH_DIR)/subscriber_cpu.prof

# View execution trace
view-trace:
	@if [ ! -f $(BENCH_DIR)/subscriber_trace.out ]; then \
		echo "⚠️  No trace found. Run 'make bench-trace' first."; \
		exit 1; \
	fi
	@echo "🔍 Opening trace viewer in browser..."
	@go tool trace $(BENCH_DIR)/subscriber_trace.out

# Clean benchmark results
clean:
	@echo "🧹 Cleaning benchmark results..."
	@rm -rf $(BENCH_DIR)
	@rm -rf benchmarks
	@rm -rf bin
	@echo "✅ Clean complete"

# Install benchstat tool (for comparison)
install-benchstat:
	@echo "📦 Installing benchstat..."
	@go install golang.org/x/perf/cmd/benchstat@latest
	@echo "✅ Done. You can now use 'make bench-compare'"
