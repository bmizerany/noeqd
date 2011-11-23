# figure out what GOROOT is supposed to be
GOROOT ?= $(shell printf 't:;@echo $$(GOROOT)\n' | gomake -f -)
include $(GOROOT)/src/Make.inc

TARG=noeqd
GOFILES=\
	main.go\

include $(GOROOT)/src/Make.cmd

VERSION=$(shell git describe --tags --always)

tar: clean $(TARG)
	tar -czf $(TARG)-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz $(TARG) README.md
