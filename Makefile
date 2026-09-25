TAG_NAME ?= $(shell git describe --tags --abbrev=0)
BUILD_NUMBER = $(shell date +%Y%m%d%H%M)

# Packaging runs through the yoga CLI (see yoga.toml); make install_deps
# installs the version go.mod builds the app with. Artifacts land in dist/<os>/.
YOGA ?= yoga
YOGA_VERSION = $(shell go list -m -f '{{.Version}}' github.com/mirzakhany/yoga)

.PHONY: install_deps
install_deps:
	go install github.com/mirzakhany/yoga/cmd/yoga@$(YOGA_VERSION)

# Use this instead of `go mod vendor`: it also restores the tree-sitter C
# sources that `go mod vendor` drops.
.PHONY: vendor
vendor:
	./build/vendor.sh

.PHONY: run
run:
	go run .

.PHONY: test
test:
	go test ./...

.PHONY: lint
lint:
	golangci-lint run -c .golangci-lint.yaml

.PHONY: fmt
fmt:
	golangci-lint fmt

.PHONY: clean
clean:
	rm -rf ./dist

# Unsigned (ad-hoc) DMGs for local testing.
.PHONY: build_macos
build_macos:
	$(YOGA) package -os darwin -arch amd64 -version $(TAG_NAME)
	$(YOGA) package -os darwin -arch arm64 -version $(TAG_NAME)

# Developer ID signed, notarized and stapled DMGs. Needs IDENTITY
# ("Developer ID Application: Name (TEAMID)") plus APPLE_ID, APPLE_TEAM_ID
# and APPLE_APP_SPECIFIC_PASSWORD (or NOTARY_KEYCHAIN_PROFILE).
.PHONY: build_macos_signed
build_macos_signed:
	@if [ -z "$(IDENTITY)" ]; then echo "ERROR: IDENTITY is not set"; exit 1; fi
	$(YOGA) package -os darwin -arch amd64 -version $(TAG_NAME) -build-number $(BUILD_NUMBER) -sign "$(IDENTITY)" -notarize
	$(YOGA) package -os darwin -arch arm64 -version $(TAG_NAME) -build-number $(BUILD_NUMBER) -sign "$(IDENTITY)" -notarize

# Universal, sandboxed, signed .pkg for App Store Connect.
.PHONY: build_appstore
build_appstore:
	@if [ -z "$(APPLE_TEAM_ID)" ]; then echo "ERROR: APPLE_TEAM_ID is not set"; exit 1; fi
	$(YOGA) package -os darwin -arch universal -format pkg \
		-version $(TAG_NAME) -build-number $(BUILD_NUMBER) \
		-sign "3rd Party Mac Developer Application: Mohsen Mirzakhani ($(APPLE_TEAM_ID))" \
		-installer-sign "3rd Party Mac Developer Installer: Mohsen Mirzakhani ($(APPLE_TEAM_ID))" \
		-entitlements build/appstore/entitlements.plist

.PHONY: upload_appstore
upload_appstore:
	@if [ -z "$(APPLE_ID)" ] || [ -z "$(APPLE_APP_SPECIFIC_PASSWORD)" ]; then \
		echo "ERROR: APPLE_ID and APPLE_APP_SPECIFIC_PASSWORD must be set"; exit 1; \
	fi
	xcrun altool --upload-app -f ./dist/darwin/chapar-macos-$(TAG_NAME)-universal.pkg -t macos -u "$(APPLE_ID)" -p "$(APPLE_APP_SPECIFIC_PASSWORD)" --verbose

.PHONY: build_linux
build_linux:
	$(YOGA) package -os linux -version $(TAG_NAME)

.PHONY: build_windows
build_windows:
	$(YOGA) package -os windows -arch amd64 -version $(TAG_NAME)
