GIT_UPDATE_INDEX    := $(shell git update-index --refresh)
GIT_VERSION         := $(shell git describe --tags --dirty 2>/dev/null || echo dev)
GO                  ?= $(shell which go)

ifneq ($(GO),)
GOENV_GOOS               := $(shell go env GOOS)
GOENV_GOARCH             := $(shell go env GOARCH)
GOOS                     ?= $(GOENV_GOOS)
GOARCH                   ?= $(GOENV_GOARCH)
GO_BUILD_SRC             := $(shell find . -name \*.go -type f ! -path ./autogen/genstatic/gen.go) go.mod go.sum

GO_GENERATE_SRC          := $(find ./config/ -type f -name \*.go ! -name \*_deepcopy.go)
GO_GENERATE_TARGET       := config/config_deepcopy.go

GO_BUILD_PREFIX          ?= dist/cerberus
GO_BUILD_EXTLDFLAGS      :=
GO_BUILD_TAGS            := static
GO_BUILD_TARGET_DEPS     :=
GO_BUILD_FLAGS           :=
GO_BUILD_LDFLAGS_OPTIMS  :=


ifeq ($(GOOS)/$(GOARCH),$(GOENV_GOOS)/$(GOENV_GOARCH))
GO_BUILD_TARGET          := $(GO_BUILD_PREFIX)
GO_BUILD_VERSION_TARGET  := $(GO_BUILD_PREFIX)-$(GIT_VERSION)
else
GO_BUILD_TARGET          := $(GO_BUILD_PREFIX)-$(GOOS)-$(GOARCH)
GO_BUILD_VERSION_TARGET  := $(GO_BUILD_PREFIX)-$(GIT_VERSION)-$(GOOS)-$(GOARCH)
endif # $(GOOS)/$(GOARCH)

ifeq ($(shell uname),Linux)
GO_BUILD_EXTLDFLAGS      += -lbsd
endif

ifneq ($(DEBUG),)
GO_BUILD_FLAGS           += -race -gcflags="all=-N -l"
else
GO_BUILD_LDFLAGS_OPTIMS  += -s -w
endif # $(DEBUG)

GO_BUILD_EXTLDFLAGS      := $(strip $(GO_BUILD_EXTLDFLAGS))
GO_BUILD_TAGS            := $(strip $(GO_BUILD_TAGS))
GO_BUILD_TARGET_DEPS     := $(strip $(GO_BUILD_TARGET_DEPS))
GO_BUILD_FLAGS           := $(strip $(GO_BUILD_FLAGS))
GO_BUILD_LDFLAGS_OPTIMS  := $(strip $(GO_BUILD_LDFLAGS_OPTIMS))
GO_BUILD_LDFLAGS         := -ldflags '$(GO_BUILD_LDFLAGS_OPTIMS) -X main.version=$(GIT_VERSION) -extldflags "$(GO_BUILD_EXTLDFLAGS)"'
endif # $(GO)

GO_BUILD_FLAGS_TARGET                := .go-build-flags
GO_CROSSBUILD_PLATFORMS              ?= linux/amd64 windows/amd64 darwin/amd64
GO_CROSSBUILD_LINUX_PLATFORMS        := $(filter linux/%,$(GO_CROSSBUILD_PLATFORMS))
GO_CROSSBUILD_FREEBSD_PLATFORMS      := $(filter freebsd/%,$(GO_CROSSBUILD_PLATFORMS))
GO_CROSSBUILD_OPENBSD_PLATFORMS      := $(filter openbsd/%,$(GO_CROSSBUILD_PLATFORMS))
GO_CROSSBUILD_WINDOWS_PLATFORMS      := $(filter windows/%,$(GO_CROSSBUILD_PLATFORMS))
GO_CROSSBUILD_DARWIN_PLATFORMS       := $(filter darwin/%,$(GO_CROSSBUILD_PLATFORMS))
GO_CROSSBUILD_LINUX_TARGET_PATTERN   := $(GO_BUILD_PREFIX)-$(GIT_VERSION)-linux-%
GO_CROSSBUILD_FREEBSD_TARGET_PATTERN := $(GO_BUILD_PREFIX)-$(GIT_VERSION)-freebsd-%
GO_CROSSBUILD_OPENBSD_TARGET_PATTERN := $(GO_BUILD_PREFIX)-$(GIT_VERSION)-openbsd-%
GO_CROSSBUILD_WINDOWS_TARGET_PATTERN := $(GO_BUILD_PREFIX)-$(GIT_VERSION)-windows-%.exe
GO_CROSSBUILD_DARWIN_TARGET_PATTERN  := $(GO_BUILD_PREFIX)-$(GIT_VERSION)-darwin-%
GO_CROSSBUILD_TARGETS                := $(patsubst linux/%,$(GO_CROSSBUILD_LINUX_TARGET_PATTERN),$(GO_CROSSBUILD_LINUX_PLATFORMS))
GO_CROSSBUILD_TARGETS                += $(patsubst freebsd/%,$(GO_CROSSBUILD_FREEBSD_TARGET_PATTERN),$(GO_CROSSBUILD_FREEBSD_PLATFORMS))
GO_CROSSBUILD_TARGETS                += $(patsubst openbsd/%,$(GO_CROSSBUILD_OPENBSD_TARGET_PATTERN),$(GO_CROSSBUILD_OPENBSD_PLATFORMS))
GO_CROSSBUILD_TARGETS                += $(patsubst windows/%,$(GO_CROSSBUILD_WINDOWS_TARGET_PATTERN),$(GO_CROSSBUILD_WINDOWS_PLATFORMS))
GO_CROSSBUILD_TARGETS                += $(patsubst darwin/%,$(GO_CROSSBUILD_DARWIN_TARGET_PATTERN),$(GO_CROSSBUILD_DARWIN_PLATFORMS))

DOCKER_IMAGE_VERSION           ?= dev
DOCKER_IMAGE_REPO              ?= quay.io/sylr/cerberus
DOCKER_BUILD_GO_FLAGS          ?=
DOCKER_BUILD_FLAGS             ?=
DOCKER_RUN_FLAGS               ?= -e INSIDE_DOCKER=1 -e DEBUG=$(DEBUG)

DOCKER_BUILD_GO_TARGET         := .docker-build-go
DOCKER_BUILD_SCRATCH_TARGET    := .docker-build-scratch
DOCKER_BUILD_TARGET            := .docker-build

# ------------------------------------------------------------------------------

.PHONY: all build

all: build

clean:
	@git clean -ndx
	@/bin/echo -n "Would you like to proceed (yes/no) ? "
	@read proceed && test "$$proceed" == "yes" && git clean -fdx
	@cd ./lib/librdkafka/ && git reset --hard

# -- tests ---------------------------------------------------------------------

.PHONY: test test-go

test: test-go

test-go:
	@go test ./...

# -- build ---------------------------------------------------------------------

.PHONY: build build-go .FORCE

$(GO_BUILD_FLAGS_TARGET) : .FORCE
	@(echo "GO_VERSION=$(shell $(GO) version)"; \
	  echo "GO_GOOS=$(GOOS)"; \
	  echo "GO_GOARCH=$(GOARCH)"; \
	  echo "GO_BUILD_TAGS=$(GO_BUILD_TAGS)"; \
	  echo "GO_BUILD_FLAGS=$(GO_BUILD_FLAGS)"; \
	  echo 'GO_BUILD_LDFLAGS=$(subst ','\'',$(GO_BUILD_LDFLAGS))') \
	    | cmp -s - $@ \
	        || (echo "GO_VERSION=$(shell $(GO) version)"; \
	            echo "GO_GOOS=$(GOOS)"; \
	            echo "GO_GOARCH=$(GOARCH)"; \
	            echo "GO_BUILD_TAGS=$(GO_BUILD_TAGS)"; \
	            echo "GO_BUILD_FLAGS=$(GO_BUILD_FLAGS)"; \
	            echo 'GO_BUILD_LDFLAGS=$(subst ','\'',$(GO_BUILD_LDFLAGS))') > $@

build: build-go

build-go: $(GO_BUILD_VERSION_TARGET) $(GO_BUILD_TARGET)

$(GO_GENERATE_TARGET): $(GO_GENERATE_SRC) | $(GO_BINDATA)
	go generate

$(GO_BUILD_TARGET): $(GO_BUILD_VERSION_TARGET)
	@(test -e $@ && unlink $@) || true
	@ln $< $@

$(GO_BUILD_VERSION_TARGET): $(GO_BUILD_SRC) $(GO_GENERATE_TARGET) $(GO_BUILD_FLAGS_TARGET) | $(GO_BUILD_TARGET_DEPS)
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build -tags $(GO_BUILD_TAGS) $(GO_BUILD_FLAGS) $(GO_BUILD_LDFLAGS) -o $@

crossbuild: $(GO_BUILD_VERSION_TARGET) $(GO_CROSSBUILD_TARGETS)

$(GO_CROSSBUILD_LINUX_TARGET_PATTERN): $(GO_BUILD_SRC) $(GO_BUILD_FLAGS_TARGET) | $(GO_GENERATE_TARGET)
	GOOS=linux GOARCH=$* $(GO) build -tags $(GO_BUILD_TAGS),crossbuild $(GO_BUILD_FLAGS) $(GO_BUILD_LDFLAGS) -o $@

$(GO_CROSSBUILD_FREEBSD_TARGET_PATTERN): $(GO_BUILD_SRC) $(GO_BUILD_FLAGS_TARGET) | $(GO_GENERATE_TARGET)
	GOOS=freebsd GOARCH=$* $(GO) build -tags $(GO_BUILD_TAGS),crossbuild $(GO_BUILD_FLAGS) $(GO_BUILD_LDFLAGS) -o $@

$(GO_CROSSBUILD_OPENBSD_TARGET_PATTERN): $(GO_BUILD_SRC) $(GO_BUILD_FLAGS_TARGET) | $(GO_GENERATE_TARGET)
	GOOS=openbsd GOARCH=$* $(GO) build -tags $(GO_BUILD_TAGS),crossbuild $(GO_BUILD_FLAGS) $(GO_BUILD_LDFLAGS) -o $@

$(GO_CROSSBUILD_WINDOWS_TARGET_PATTERN): $(GO_BUILD_SRC) $(GO_BUILD_FLAGS_TARGET) | $(GO_GENERATE_TARGET)
	GOOS=windows GOARCH=$* $(GO) build -tags $(GO_BUILD_TAGS),crossbuild $(GO_BUILD_FLAGS) $(GO_BUILD_LDFLAGS) -o $@

$(GO_CROSSBUILD_DARWIN_TARGET_PATTERN): $(GO_BUILD_SRC) $(GO_BUILD_FLAGS_TARGET) | $(GO_GENERATE_TARGET)
	GOOS=darwin GOARCH=$* $(GO) build -tags $(GO_BUILD_TAGS),crossbuild $(GO_BUILD_FLAGS) $(GO_BUILD_LDFLAGS) -o $@

# -- tools ---------------------------------------------------------------------

.PHONY: git-hooks

git-hooks:
	@{ test -e contrib -a -e .git && cp contrib/git/hooks/pre-commit .git/hooks/ } || true

# -- docker --------------------------------------------------------------------

.PHONY: docker-build-go-image docker-build docker-push docker-test-go docker-test docker-test-go docker-clean-dist

docker-build-go-image: docker-build-react-image $(DOCKER_BUILD_GO_TARGET)

$(DOCKER_BUILD_GO_TARGET): $(GO_BUILD_SRC)
	docker build $(DOCKER_BUILD_FLAGS) $(DOCKER_BUILD_GO_FLAGS) -t "cerberus-go:$(DOCKER_IMAGE_VERSION)" -f Dockerfile-go .
	@touch $@

docker-build: $(DOCKER_BUILD_TARGET)

$(DOCKER_BUILD_TARGET): $(DOCKER_BUILD_REACT_TARGET) $(DOCKER_BUILD_GO_TARGET)
	docker build $(DOCKER_BUILD_FLAGS) -t "cerberus:$(DOCKER_IMAGE_VERSION)" -f Dockerfile .
	docker tag "cerberus:$(DOCKER_IMAGE_VERSION)" "$(DOCKER_IMAGE_REPO):$(GIT_VERSION)"
	@touch $@

docker-build-scratch:
	docker build $(DOCKER_BUILD_FLAGS) -t "cerberus:$(DOCKER_IMAGE_VERSION)-stratch" -f Dockerfile-scratch .
	docker tag "cerberus:$(DOCKER_IMAGE_VERSION)-stratch" "$(DOCKER_IMAGE_REPO):$(GIT_VERSION)-scratch"

docker-crossbuild: docker-build-react-image docker-build-go-image
	docker run $(DOCKER_RUN_FLAGS) -v $(CURDIR)/dist:/go/src/github.com/sylr/cerberus/dist -t "cerberus-go:$(DOCKER_IMAGE_VERSION)" make crossbuild

docker-push:
	docker push "$(DOCKER_IMAGE_REPO):$(GIT_VERSION)"

docker-test-go: docker-build-go-image
	docker run $(DOCKER_RUN_FLAGS) -v $(CURDIR)/dist:/go/src/github.com/sylr/cerberus/dist -t "cerberus-go:$(DOCKER_IMAGE_VERSION)" make test

docker-test: docker-test-go

docker-clean-dist: docker-build-go-image
	docker run $(DOCKER_RUN_FLAGS) -v $(CURDIR)/dist:/go/src/github.com/sylr/cerberus/dist -t "cerberus-go:$(DOCKER_IMAGE_VERSION)" rm -rf dist/*
