.PHONY: build_macos
build_macos:
	@echo "Building Macos..."
	gogio -icon=./assets/appicon.png -target=macos -arch=amd64 -o ./dist/amd64/Chapar.app .
	gogio -icon=./assets/appicon.png -target=macos -arch=arm64 -o ./dist/arm64/Chapar.app .
	codesign --force --deep --sign - ./dist/amd64/Chapar.app
	codesign --force --deep --sign - ./dist/arm64/Chapar.app

.PHONY: build
build: build_macos

.PHONY: run
run:
	@echo "Running..."
	go run .

.PHONY: clean
clean:
	@echo "Cleaning..."
	rm -rf ./Chapar.app

.PHONY: install_deps
install_deps:
	go install gioui.org/cmd/gogio@latest

.PHONY: lint
lint:
	docker run --rm \
		-e CGO_ENABLED=1 \
		-v $(PWD):/app \
		-w /app chapar/builder:0.1.3 \
		 golangci-lint -c .golangci-lint.yaml run --timeout 5m

.PHONY: test
test:
	docker run --rm \
		-e CGO_ENABLED=1 \
		-v $(PWD):/app \
		-w /app chapar/builder:0.1.3 \
		go test -v ./...
