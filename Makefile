VERSION := ${shell cat ./VERSION}
IMPORT_PATH := github.com/hms-dbmi/vault-getter
IGNORED_PACKAGES := /vendor/ # space separated patterns
VERSION_FLAGS    := -ldflags='-X "main.Version=$(VERSION)"'

Q := $(if $V,,@)
V := 1 # print commands and build progress by default

.PHONY: all
all: test build

.PHONY: hello
build: .GOPATH/.ok
	$Q go install $(if $V,-v) $(VERSION_FLAGS) $(IMPORT_PATH)/cmd/vault_getter

.PHONY: clean
clean:
	$Q rm -rf .GOPATH/.ok .GOPATH bin/vault_getter bin vault-getter

.PHONY: test
test: .GOPATH/.ok
	$Q go vet $(allpackages)
	$Q go test -race $(allpackages)

.PHONY: list
list: .GOPATH/.ok
	@echo $(allpackages)

lint: format
	$Q go vet $(allpackages)

format: .GOPATH/.ok
	$Q echo $(_allpackages)
	$Q find .GOPATH/src/$(IMPORT_PATH)/ -iname \*.go | grep -v -e "^$$" $(addprefix -e ,$(IGNORED_PACKAGES)) | xargs goimports -w

# cd into the GOPATH to workaround ./... not following symlinks
_allpackages = $(shell ( cd "$(CURDIR)"/.GOPATH/src/$(IMPORT_PATH) && \
							 GOPATH="$(CURDIR)"/.GOPATH go list ./... 2>&1 1>&3 | \
							 grep -v -e "^$$" $(addprefix -e ,$(IGNORED_PACKAGES)) 1>&2 ) 3>&1 | \
							 grep -v -e "^$$" $(addprefix -e ,$(IGNORED_PACKAGES)))

# memoize allpackages, so that it's executed only once and only if used
allpackages = $(if $(__allpackages),,$(eval __allpackages := $$(_allpackages)))$(__allpackages)

SYS_GOPATH = $(shell echo $$GOPATH)

.GOPATH/.ok:
	$Q ln -s "$(SYS_GOPATH)" .GOPATH
	$Q mkdir -p "$(dir .GOPATH/src/$(IMPORT_PATH))"
	$Q ln -s "$(CURDIR)" ".GOPATH/src/$(IMPORT_PATH)"
	$Q ln -s .GOPATH/bin bin
	$Q touch $@
