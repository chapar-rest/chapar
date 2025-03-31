TAG_NAME?=$(shell git describe --tags --abbrev=0)
APP_NAME="Chapar"

.PHONY: build_macos_app
build_macos_app:
	@echo "Building Macos..."
	gogio -ldflags="-X main.serviceVersion=$(TAG_NAME)" -appid=rest.chapar.app -icon=./build/appicon.png -target=macos -arch=amd64 -o ./dist/amd64/Chapar.app .
	gogio -ldflags="-X main.serviceVersion=$(TAG_NAME)" -appid=rest.chapar.app -icon=./build/appicon.png -target=macos -arch=arm64 -o ./dist/arm64/Chapar.app .
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
build_macos_dmg:
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

.PHONY: build_macos_signed
build_macos_signed:
	@echo "Building and signing macOS app..."
	@if [ -z "$(APPLE_TEAM_ID)" ]; then \
		echo "ERROR: APPLE_TEAM_ID environment variable is not set"; \
		exit 1; \
	fi
	@if [ -z "$(IDENTITY)" ]; then \
		echo "ERROR: IDENTITY environment variable is not set (e.g., 'Developer ID Application: Your Name (TEAMID)')"; \
		exit 1; \
	fi

	# Build apps
	gogio -ldflags="-X main.serviceVersion=$(TAG_NAME)" -appid=rest.chapar.app -icon=./build/appicon.png -target=macos -arch=amd64 -o ./dist/amd64/Chapar.app .
	gogio -ldflags="-X main.serviceVersion=$(TAG_NAME)" -appid=rest.chapar.app -icon=./build/appicon.png -target=macos -arch=arm64 -o ./dist/arm64/Chapar.app .

	# Sign apps with Developer ID
	codesign --force --options runtime --deep -vvv --sign "$(IDENTITY)" ./dist/amd64/Chapar.app
	codesign --force --options runtime --deep --sign "$(IDENTITY)" ./dist/arm64/Chapar.app

	# Verify signing
	codesign -dvv ./dist/amd64/Chapar.app
	codesign -dvv ./dist/arm64/Chapar.app

	@echo "Apps built and signed. To notarize, run:"
	@echo "  ditto -c -k --keepParent ./dist/amd64/Chapar.app ./dist/Chapar-amd64.zip"
	@echo "  xcrun notarytool submit ./dist/Chapar-amd64.zip --apple-id \"\$$APPLE_ID\" --password \"\$$APPLE_APP_SPECIFIC_PASSWORD\" --team-id \"\$$APPLE_TEAM_ID\" --wait"
	@echo "  xcrun stapler staple ./dist/amd64/Chapar.app"

	@echo "Then create DMG with 'make build_macos_dmg'"

.PHONY: notarize_macos
notarize_macos:
	@echo "Notarizing macOS apps..."
	@if [ -z "$(APPLE_ID)" ] || [ -z "$(APPLE_TEAM_ID)" ] || [ -z "$(APPLE_APP_SPECIFIC_PASSWORD)" ]; then \
		echo "ERROR: One or more required environment variables are not set:"; \
		echo "  - APPLE_ID"; \
		echo "  - APPLE_TEAM_ID"; \
		echo "  - APPLE_APP_SPECIFIC_PASSWORD"; \
		exit 1; \
	fi

	# Create zip archives for notarization
	ditto -c -k --keepParent ./dist/amd64/Chapar.app ./dist/Chapar-amd64.zip
	ditto -c -k --keepParent ./dist/arm64/Chapar.app ./dist/Chapar-arm64.zip

	# Submit for notarization
	xcrun notarytool submit ./dist/Chapar-amd64.zip --apple-id "$(APPLE_ID)" --password "$(APPLE_APP_SPECIFIC_PASSWORD)" --team-id "$(APPLE_TEAM_ID)" --wait
	xcrun notarytool submit ./dist/Chapar-arm64.zip --apple-id "$(APPLE_ID)" --password "$(APPLE_APP_SPECIFIC_PASSWORD)" --team-id "$(APPLE_TEAM_ID)" --wait

	@echo "Notarization complete. Now run 'make build_macos_dmg' to create DMG files."

.PHONY: build_appstore
build_appstore:
	@echo "Building for App Store..."
	@if [ -z "$(APPLE_TEAM_ID)" ]; then \
		echo "ERROR: APPLE_TEAM_ID environment variable is not set"; \
		exit 1; \
	fi
	@if [ -z "$(TAG_NAME)" ]; then \
		echo "ERROR: TAG_NAME environment variable is not set"; \
		exit 1; \
	fi

	./build/build_appstore.sh $(TAG_NAME)


.PHONY: upload_appstore
upload_appstore:
	@echo "Uploading to App Store Connect..."
	@if [ -z "$(APPLE_ID)" ] || [ -z "$(APPLE_APP_SPECIFIC_PASSWORD)" ] || [ -z "$(TAG_NAME)" ]; then \
		echo "ERROR: One or more required environment variables are not set:"; \
		echo "  - APPLE_ID"; \
		echo "  - APPLE_APP_PASSWORD"; \
		echo "  - VERSION"; \
		exit 1; \
	fi

	xcrun altool --upload-app -f ./dist/Chapar-$(TAG_NAME).pkg -t macos -u "$(APPLE_ID)" -p "$(APPLE_APP_SPECIFIC_PASSWORD)" --verbose


.PHONY: build_windows
build_windows:
	@echo "Building Windows..."
	cp build\appicon.png .
	gogio -ldflags="-X main.serviceVersion=${TAG_NAME}" -target=windows -arch=amd64 -o "dist\amd64\Chapar.exe" .
	gogio -ldflags="-X main.serviceVersion=${TAG_NAME}" -target=windows -arch=386 -o "dist\i386\Chapar.exe" .
	gogio -ldflags="-X main.serviceVersion=${TAG_NAME}" -target=windows -arch=arm64 -o "dist\arm64\Chapar.exe" .
	rm *.syso
	powershell -Command "Compress-Archive -Path dist\amd64\Chapar.exe -Destination dist\chapar-windows-${TAG_NAME}-amd64.zip"
	powershell -Command "Compress-Archive -Path dist\i386\Chapar.exe -Destination dist\chapar-windows-${TAG_NAME}-i386.zip"
	powershell -Command "Compress-Archive -Path dist\arm64\Chapar.exe -Destination dist\chapar-windows-${TAG_NAME}-arm64.zip"
	rm -rf .\dist\amd64
	rm -rf .\dist\i386
	rm -rf .\dist\arm64

.PHONY: build_linux
build_linux:
	@echo "Building Linux amd64..."
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-X main.serviceVersion=$(TAG_NAME)" -o ./dist/amd64/chapar .
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
		-w /app chapar/builder:0.1.5 \
		 golangci-lint -c .golangci-lint.yaml run --timeout 5m

.PHONY: test
test:
	docker run --rm \
		-e CGO_ENABLED=1 \
		-v $(PWD):/app \
		-w /app chapar/builder:0.1.5 \
		go test -v ./...
