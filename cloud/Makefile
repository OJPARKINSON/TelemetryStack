.PHONY: dev fix-podman check-podman build-server build-server-prod

build-server:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ./bin/server ./cmd/api

build-server-prod:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/server ./cmd/api

# Start wrangler dev, fixing podman runtime if needed
dev: build-server check-podman
	pnpm dev

build-frontend:
	cd ../dashboard/ && pnpm run build

deploy: build-frontend build-server-prod
	pnpm run deploy

# Check if podman VM has runc configured; fix automatically if not
check-podman:
	@runtime=$$(podman info --format '{{.Host.OCIRuntime.Name}}' 2>/dev/null) && \
	if [ "$$runtime" = "runc" ]; then \
		echo "podman: runc runtime OK"; \
	else \
		echo "podman: runtime is '$$runtime', need runc — running fix-podman..."; \
		$(MAKE) fix-podman; \
	fi

# Install runc in podman VM and set it as default runtime.
# Needed because workerd sends MemorySwappiness in container create requests,
# which crun rejects on cgroupv2. runc silently ignores it.
fix-podman:
	@echo "Installing runc in podman VM..."
	podman machine ssh -- "\
		curl -fsSL -o /usr/local/bin/runc \
			https://github.com/opencontainers/runc/releases/download/v1.2.5/runc.$$(podman machine ssh -- uname -m | sed 's/aarch64/arm64/;s/x86_64/amd64/') && \
		chmod +x /usr/local/bin/runc && \
		sudo mkdir -p /etc/containers && \
		printf '[engine]\nruntime = \"runc\"\n' | sudo tee /etc/containers/containers.conf > /dev/null"
	@echo "Restarting podman machine..."
	podman machine stop && podman machine start
	@echo "Done. Runtime: $$(podman info --format '{{.Host.OCIRuntime.Name}}')"
