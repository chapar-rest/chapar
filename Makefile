BUILDNAME=chapar
SHORT_SHA?=$(shell git rev-parse --short HEAD)
VERSION="v0.1.0"-$(SHORT_SHA)

.PHONY: docker_builder
docker_builder:
	@echo "Building builder image..."
	docker build -t chapar-builder -f builder/Dockerfile .

.PHONY: run_docker_builder
run_docker_builder: docker_builder
	@echo "Building builder image..."
	docker run -it --rm \
		--name chapar-builder-instance \
		-e BUILDOS=$(BUILDOS) \
		-e BUILDARCH=$(BUILDARCH) \
		-e CGO_ENABLED=1 \
		-e BUILDNAME=$(BUILDNAME)-$(BUILDOS)-$(BUILDARCH)-$(VERSION) \
		--mount type=bind,source="$(PWD)",target=/app \
		chapar-builder \
		sh -c "GOOS=$$BUILDOS GOARCH=$$BUILDARCH go build -mod vendor -o /app/output/$$BUILDNAME -trimpath -ldflags=-buildid="


.PHONY: docker_build
docker_build:
	make BUILDOS=darwin BUILDARCH=arm64 run_docker_builder


.PHONY: build
build:
	@echo "Building..."
	gogio -icon=appicon.png -target=macos -o ./Chapar.app .

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
