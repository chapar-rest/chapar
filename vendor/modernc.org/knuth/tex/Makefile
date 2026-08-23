# Copyright 2023 The Knuth Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

.PHONY:	all clean edit editor generate fpc

all:

clean:
	rm -f cpu.test mem.test *.out
	go clean

edit:
	@touch log
	@if [ -f "Session.vim" ]; then gvim -S & else gvim -p Makefile *.go & fi

editor:
	gofmt -l -s -w .
	go test 2>&1 | tee log-editor

generate:
	go run generate.go
	go build -v ./...
	weave tex.web tex.ch
	rm -f tex.pas tex.tex CONTENTS.tex tex.log *.p internal/trip/tex.pas
