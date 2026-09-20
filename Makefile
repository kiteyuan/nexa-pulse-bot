.PHONY: test vet lint frontend

test:
	go test ./...

vet:
	go vet ./...

lint: vet
	golangci-lint run ./...

frontend:
	cd web/admin && npm ci && npm run build
	cd web/public && npm ci && npm run build

dev:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/dev.ps1
