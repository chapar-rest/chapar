.PHONY: build_macos
build_macos:
	@echo "Building Macos..."
	gogio -icon=./build/appicon.png -target=macos -arch=amd64 -o ./dist/amd64/Chapar.app .
	gogio -icon=./build/appicon.png -target=macos -arch=arm64 -o ./dist/arm64/Chapar.app .
	codesign --force --deep --sign - ./dist/amd64/Chapar.app
	codesign --force --deep --sign - ./dist/arm64/Chapar.app
	tar -cJf ./dist/Chapar_macos_amd64.tar.xz ./dist/amd64/Chapar.app
	tar -cJf ./dist/Chapar_macos_arm64.tar.xz ./dist/arm64/Chapar.app
	rm -rf ./dist/amd64
	rm -rf ./dist/arm64

.PHONY: build_windows
build_windows:
	@echo "Building Windows..."
	gogio -icon=./build/appicon.png -buildmode=archive -target=windows -arch=amd64 -o ./dist/amd64/Chapar.exe .
	gogio -icon=./build/appicon.png -buildmode=archive -target=windows -arch=386 -o ./dist/i386/Chapar.exe .
	rm *.syso
	tar -cJf ./dist/Chapar_windows_amd64.tar.xz ./dist/amd64/Chapar.exe
	tar -cJf ./dist/Chapar_windows_i386.tar.xz ./dist/i386/Chapar.exe
	rm -rf ./dist/amd64
	rm -rf ./dist/i386

.PHONY: build_linux
build_linux:
	@echo "Building Linux..."
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o ./dist/amd64/chapar .
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -o ./dist/arm64/chapar .
	tar -cJf ./dist/Chapar_linux_amd64.tar.xz ./dist/amd64/chapar ./build/desktop-assets ./build/install-linux.sh ./build/appicon.png ./LICENSE
	tar -cJf ./dist/Chapar_linux_arm64.tar.xz ./dist/arm64/chapar ./build/desktop-assets ./build/install-linux.sh ./build/appicon.png ./LICENSE
	rm -rf ./dist/amd64
	rm -rf ./dist/arm64

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
