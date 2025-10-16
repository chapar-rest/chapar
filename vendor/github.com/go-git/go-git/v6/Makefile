# General
WORKDIR = $(PWD)

# Go parameters
GOCMD = go
GOTEST = $(GOCMD) test 

# Git config
GIT_VERSION ?=
GIT_DIST_PATH ?= $(PWD)/.git-dist
GIT_REPOSITORY = http://github.com/git/git.git

# Coverage
COVERAGE_REPORT = coverage.out
COVERAGE_MODE = count

# Defines the maximum time each fuzz target will be executed for.
FUZZ_TIME ?= 10s
FUZZ_PKGS = $(shell grep -r --include='**_test.go' --files-with-matches 'func Fuzz' . | xargs -I{} dirname {})

build-git:
	@if [ -f $(GIT_DIST_PATH)/git ]; then \
		echo "nothing to do, using cache $(GIT_DIST_PATH)"; \
	else \
		git clone $(GIT_REPOSITORY) -b $(GIT_VERSION) --depth 1 --single-branch $(GIT_DIST_PATH); \
		cd $(GIT_DIST_PATH); \
		make configure; \
		./configure; \
		make all; \
	fi

test:
	@echo "running against `git version`"; \
	$(GOTEST) -race ./...
	$(GOTEST) -v _examples/common_test.go _examples/common.go --examples

test-coverage:
	@echo "running against `git version`"; \
	echo "" > $(COVERAGE_REPORT); \
	$(GOTEST) -coverprofile=$(COVERAGE_REPORT) -coverpkg=./... -covermode=$(COVERAGE_MODE) ./...

clean:
	rm -rf $(GIT_DIST_PATH)

fuzz:
	@for path in $(FUZZ_PKGS); do \
		go test -fuzz=Fuzz -fuzztime=$(FUZZ_TIME) $$path; \
	done
