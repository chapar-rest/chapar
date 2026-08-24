# Copyright 2023 The Knuth Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

.PHONY:	all clean edit editor test

all:
	go generate -v ./... || true # https://github.com/golang/go/issues/13440
	# grep -nir 'knuthw\|\<2022\>\|pdftex' || true
	gofmt -l -s -w .
	go build -v ./...
	go test -count=1 ./... 2>&1 | tee log-editor
	go install -v ./cmd...
	# golint ./...
	staticcheck

clean:
	rm -f cpu.test mem.test *.out
	go clean

edit:
	@touch log
	@if [ -f "Session.vim" ]; then gvim -S & else gvim -p Makefile go.mod builder.json *.go & fi

editor:
	@grep -nir 'knuthw\|\<2022\>\|pdftex' || true
	@find -name \*.pdf -or -name \*.pas
	gofmt -l -s -w *.go
	go test 2>&1 | tee log-editor

test:
	go test -v -count=1 ./...
