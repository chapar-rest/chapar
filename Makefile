.PHONY: build_macos
build_macos:
	@echo "Building Macos..."
	gogio -icon=appicon.png -target=macos -arch=amd64 -o ./dist/amd64/Chapar.app .
	gogio -icon=appicon.png -target=macos -arch=arm64 -o ./dist/arm64/Chapar.app .
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
