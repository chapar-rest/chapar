TAG_NAME?=$(shell git describe --tags)
APP_NAME="Chapar"

.PHONY: build_macos_app
build_macos_app:
	@echo "Building Macos..."
	gogio -icon=./build/appicon.png -target=macos -arch=amd64 -o ./dist/amd64/Chapar.app .
	gogio -icon=./build/appicon.png -target=macos -arch=arm64 -o ./dist/arm64/Chapar.app .
	codesign --force --deep --sign - ./dist/amd64/Chapar.app
	codesign --force --deep --sign - ./dist/arm64/Chapar.app

.PHONY: build_macos
build_macos: build_macos_app
	@echo "Building Macos..."
	tar -cJf ./dist/Chapar_macos_amd64.tar.xz --directory=./dist/amd64 Chapar.app
	tar -cJf ./dist/Chapar_macos_arm64.tar.xz --directory=./dist/arm64 Chapar.app
	rm -rf ./dist/amd64
	rm -rf ./dist/arm64

.PHONY: build_macos_dmg
build_macos_dmg: build_macos_app
	@echo "Building Macos DMG..."
	rm -rf ./dist/chapar-macos-$(TAG_NAME)-amd64.dmg
	rm -rf ./dist/chapar-macos-$(TAG_NAME)-arm64.dmg
	create-dmg \
	  --volname "Chapar Installer" \
	  --volicon "./build/appicon.icns" \
	  --background "./build/chapar-installer-bk.png" \
	  --window-pos 300 300 \
	  --window-size 500 350 \
	  --icon-size 100 \
	  --icon "Chapar.app" 125 150 \
	  --hide-extension "Chapar.app" \
	  --app-drop-link 375 150 \
	  "./dist/chapar-macos-$(TAG_NAME)-arm64.dmg" \
	  "./dist/arm64/Chapar.app"

	create-dmg \
	  --volname "Chapar Installer" \
	  --volicon "./build/appicon.icns" \
	  --background "./build/chapar-installer-bk.png" \
	  --window-pos 300 300 \
	  --window-size 500 350 \
	  --icon-size 100 \
	  --icon "Chapar.app" 125 150 \
	  --hide-extension "Chapar.app" \
	  --app-drop-link 375 150 \
	  "./dist/chapar-macos-$(TAG_NAME)-amd64.dmg" \
	  "./dist/amd64/Chapar.app"


.PHONY: build_windows
build_windows:
	@echo "Building Windows..."
	cp build\appicon.png .
	gogio -target=windows -arch=amd64 -o dist\amd64\Chapar.exe .
	gogio -target=windows -arch=386 -o dist\i386\Chapar.exe .
	rm *.syso
	powershell -Command "Compress-Archive -Path dist\amd64\Chapar.exe -Destination dist\chapar-windows-$(TAG_NAME)-amd64.zip"
	powershell -Command "Compress-Archive -Path dist\i386\Chapar.exe -Destination dist\chapar-windows-$(TAG_NAME)-i386.zip"
	rm -rf .\dist\amd64
	rm -rf .\dist\i386

.PHONY: build_linux
build_linux:
	@echo "Building Linux amd64..."
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o ./dist/amd64/chapar .
	cp ./build/install-linux.sh ./dist/amd64
	cp ./build/appicon.png ./dist/amd64
	cp ./LICENSE ./dist/amd64
	cp -r ./build/desktop-assets ./dist/amd64
	tar -cJf ./dist/chapar-linux-$(TAG_NAME)-amd64.tar.xz --directory=./dist/amd64 chapar desktop-assets install-linux.sh appicon.png ./LICENSE
	rm -rf ./dist/amd64

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
