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
