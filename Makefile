export CGO_ENABLED=0
export GOOS=linux
export GOARCH=amd64
export VERSION=(unknown)
GO := go
ENV ?= dev
LDFLAGS ?= -X main.version=$(VERSION)
BUILDFLAGS ?= -a -ldflags '$(LDFLAGS)'
APPSOURCES := $(wildcard internal/*/*.go cmd/*.go calendar/*.go calendar/*/*.go storage/*.go storage/*/*.go ical/*.go)
PROJECT_NAME := $(shell basename $(PWD))
CALENDARS ?= "tl pfw"

M4 = /usr/bin/m4

DESTDIR = /
INSTALL_PREFIX = usr/local/
USERUNITDIR = lib/systemd/system
LIBDIR = var/lib

BIN_DIR ?= $(DESTDIR)$(INSTALL_PREFIX)bin
DATA_DIR ?= $(DESTDIR)$(LIBDIR)/$(PROJECT_NAME)

ifneq ($(ENV), dev)
	LDFLAGS += -s -w -extldflags "-static"
endif

ifeq ($(shell git describe --always > /dev/null 2>&1 ; echo $$?), 0)
export VERSION = $(shell git describe --always --dirty="-git")
endif
ifeq ($(shell git describe --tags > /dev/null 2>&1 ; echo $$?), 0)
export VERSION = $(shell git describe --tags)
endif

BUILD := $(GO) build $(BUILDFLAGS)
TEST := $(GO) test $(BUILDFLAGS)

.PHONY: all ecalctl ecalserver clean test coverage install uninstall units

all: ecalctl ecalserver units

ecalctl: bin/ecalctl
bin/ecalctl: go.mod cli/ecalctl/main.go $(APPSOURCES)
	$(BUILD) -tags $(ENV) -o $@ ./cli/ecalctl/main.go

ecalserver: bin/ecalserver
bin/ecalserver: go.mod cli/ecalserver/main.go $(APPSOURCES)
	$(BUILD) -tags $(ENV) -o $@ ./cli/ecalserver/main.go

clean:
	-$(RM) bin/*
	-$(RM) units/*.service

test: TEST_TARGET := ./...
test:
	$(TEST) $(TEST_FLAGS) $(TEST_TARGET)

coverage: TEST_TARGET := .
coverage: TEST_FLAGS += -covermode=count -coverprofile $(PROJECT_NAME).coverprofile
coverage: test

units: units/ecalevents.service units/ecalserver.service units/ecaltooter.service

units/ecalevents.service: units/ecalevents.service.in
	$(M4) -DCALENDARS=$(CALENDARS) -DDATA_DIR=$(DATA_DIR) -DBIN_DIR=$(BIN_DIR) $< >$@
units/ecaltooter.service: units/ecaltooter.service.in
	$(M4) -DCALENDARS=$(CALENDARS) -DDATA_DIR=$(DATA_DIR) -DBIN_DIR=$(BIN_DIR) $< >$@
units/ecalserver.service: units/ecalserver.service.in
	$(M4) -DDATA_DIR=$(DATA_DIR) -DBIN_DIR=$(BIN_DIR) $< >$@

install: units ecalctl ecalserver
	test -d $(DESTDIR)$(INSTALL_PREFIX)$(USERUNITDIR)/ || mkdir -p $(DESTDIR)$(INSTALL_PREFIX)$(USERUNITDIR)/

	cp units/ecalevents.service $(DESTDIR)$(INSTALL_PREFIX)$(USERUNITDIR)/
	cp units/ecalevents.timer $(DESTDIR)$(INSTALL_PREFIX)$(USERUNITDIR)/
	cp units/ecalserver.service $(DESTDIR)$(INSTALL_PREFIX)$(USERUNITDIR)/
	#cp units/ecaltooter.service $(DESTDIR)$(INSTALL_PREFIX)$(USERUNITDIR)/
	#cp units/ecaltooter.timer $(DESTDIR)$(INSTALL_PREFIX)$(USERUNITDIR)/

uninstall:
	$(RM) $(DESTDIR)$(INSTALL_PREFIX)$(USERUNITDIR)/ecalevents.service
	$(RM) $(DESTDIR)$(INSTALL_PREFIX)$(USERUNITDIR)/ecalevents.timer
	$(RM) $(DESTDIR)$(INSTALL_PREFIX)$(USERUNITDIR)/ecalserver.service
	#$(RM) $(DESTDIR)$(INSTALL_PREFIX)$(USERUNITDIR)/ecaltooter.service
	#$(RM) $(DESTDIR)$(INSTALL_PREFIX)$(USERUNITDIR)/ecaltooter.timer
