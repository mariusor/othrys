export CGO_ENABLED=0
export GOOS=linux
export GOARCH=amd64
export VERSION=(unknown)
GO := go
ENV ?= dev
LDFLAGS ?= -X main.version=$(VERSION)
BUILDFLAGS ?= -a -ldflags '$(LDFLAGS)'
APPSOURCES := $(wildcard internal/*/*.go cmd/*/*.go calendar/*.go calendar/*/*.go storage/*.go storage/*/*.go ical/*.go)
PROJECT_NAME := othrys
CALENDARS ?= "tl pfw"

M4 = /usr/bin/m4

DESTDIR = /
INSTALL_PREFIX = usr/local/
USERUNITDIR = lib/systemd/user
LIBDIR = var/lib

BIN_DIR ?= $(DESTDIR)$(INSTALL_PREFIX)bin
DATA_DIR ?= %h/.local/share/$(PROJECT_NAME)

BIN_CTL = $(PROJECT_NAME)ctl
BIN_ICAL = $(PROJECT_NAME)ical

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

.PHONY: all $(BIN_CTL) $(BIN_ICAL) clean test coverage install uninstall units download

all: $(BIN_CTL) $(BIN_ICAL) units

download:
	$(GO) mod download all
	$(GO) mod tidy

$(BIN_CTL): bin/$(BIN_CTL)
bin/$(BIN_CTL): go.mod cmd $(APPSOURCES)
	$(BUILD) -tags $(ENV) -o $@ ./cmd/$(BIN_CTL)/main.go

$(BIN_ICAL): bin/$(BIN_ICAL)
bin/$(BIN_ICAL): go.mod cmd $(APPSOURCES)
	$(BUILD) -tags $(ENV) -o $@ ./cmd/$(BIN_ICAL)/main.go

clean:
	$(RM) bin/*
	$(RM) units/*.service

test: TEST_TARGET := ./...
test:
	$(TEST) $(TEST_FLAGS) $(TEST_TARGET)

coverage: TEST_TARGET := .
coverage: TEST_FLAGS += -covermode=count -coverprofile $(PROJECT_NAME).coverprofile
coverage: test

units: $(patsubst units/%.service.in, units/%.service, $(wildcard units/*.service.in))

units/%.service: units/%.service.in
	$(M4) -DCALENDARS=$(CALENDARS) -DDATA_DIR=$(DATA_DIR) -DBIN_DIR=$(BIN_DIR) \
		-DBIN_CTL=$(BIN_CTL) -DBIN_ICAL=$(DBIN_ICAL) $< >$@

mod_tidy:
	$(GO) mod tidy

install: units $(BIN_CTL) $(BIN_ICAL)
	test -d $(DESTDIR)$(INSTALL_PREFIX)$(USERUNITDIR)/ || mkdir -p $(DESTDIR)$(INSTALL_PREFIX)$(USERUNITDIR)/
	test -d $(DATA_DIR)/ || mkdir -p $(DATA_DIR)/

	install ./bin/$(BIN_CTL) $(BIN_DIR)
	install ./bin/$(BIN_ICAL) $(BIN_DIR)
	install -m 644 units/events.service $(DESTDIR)$(INSTALL_PREFIX)$(USERUNITDIR)/$(PROJECT_NAME)-events.service
	install -m 644 units/events.timer $(DESTDIR)$(INSTALL_PREFIX)$(USERUNITDIR)/$(PROJECT_NAME)-events.timer
	install -m 644 units/server.service $(DESTDIR)$(INSTALL_PREFIX)$(USERUNITDIR)/$(PROJECT_NAME)-server.service
	#install -m 644 units/tooter.service $(DESTDIR)$(INSTALL_PREFIX)$(USERUNITDIR)/$(PROJECT_NAME)-tooter.service
	#install -m 644 units/tooter.timer $(DESTDIR)$(INSTALL_PREFIX)$(USERUNITDIR)/$(PROJECT_NAME)-tooter.timer

uninstall:
	$(RM) $(BIN_DIR)/$(BIN_CTL)
	$(RM) $(BIN_DIR)/$(BIN_ICAL)
	$(RM) $(DESTDIR)$(INSTALL_PREFIX)$(USERUNITDIR)/$(PROJECT_NAME)-events.service
	$(RM) $(DESTDIR)$(INSTALL_PREFIX)$(USERUNITDIR)/$(PROJECT_NAME)-events.timer
	$(RM) $(DESTDIR)$(INSTALL_PREFIX)$(USERUNITDIR)/$(PROJECT_NAME)-server.service
	-#$(RM) $(DESTDIR)$(INSTALL_PREFIX)$(USERUNITDIR)/$(PROJECT_NAME)-tooter.service
	-#$(RM) $(DESTDIR)$(INSTALL_PREFIX)$(USERUNITDIR)/$(PROJECT_NAME)-tooter.timer
