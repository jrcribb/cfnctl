check: test lint vet

.PHONY: test
test:
	go test -cover -race -v ./...

.PHONY: lint
lint:
	golint ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: build
build:
	CGO_ENABLED=0 go build ./cmd/cfnctl

# ==================================================================================== #  
#  QUALITY CONTROL
# ==================================================================================== #

.PHONY: audit
audit:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
