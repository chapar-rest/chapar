.PHONY: build_macos
build_macos:
	@echo "Building Macos..."
	gogio -icon=./build/appicon.png -target=macos -arch=amd64 -o ./dist/amd64/Chapar.app .
	gogio -icon=./build/appicon.png -target=macos -arch=arm64 -o ./dist/arm64/Chapar.app .
	codesign --force --deep --sign - ./dist/amd64/Chapar.app
	codesign --force --deep --sign - ./dist/arm64/Chapar.app

.PHONY: build_windows
build_windows:
	@echo "Building Windows..."
	gogio -icon=./build/appicon.png -buildmode=archive -target=windows -arch=amd64 -o ./dist/amd64/Chapar.exe .
	gogio -icon=./build/appicon.png -buildmode=archive -target=windows -arch=386 -o ./dist/386/Chapar.exe .
	rm *.syso

.PHONY: build_linux
build_linux:
	@echo "Building Linux..."
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o ./dist/amd64/Chapar .
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -o ./dist/arm64/Chapar .
	tar -cJf ./dist/amd64/Chapar_amd64.tar.xz ./dist/amd64/Chapar ./build/desktop-assets ./build/install-linux.sh ./build/appicon.png ./LICENSE
	tar -cJf ./dist/arm64/Chapar_arm64.tar.xz ./dist/arm64/Chapar ./build/desktop-assets ./build/install-linux.sh ./build/appicon.png ./LICENSE


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
