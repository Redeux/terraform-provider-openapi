VERSION  = $(shell cat ./version)
RELEASE_TAG?=v$(VERSION)
CURRENT_RELEASE_TAG?=$(shell git describe --abbrev=0 --tags)
NEW_RELEASE_VERSION_VALIDATION?=$(shell ./scripts/semver.sh $(RELEASE_TAG) $(CURRENT_RELEASE_TAG))

COMMIT :=$(shell git rev-parse --verify --short HEAD)
DATE :=$(shell date +'%FT%TZ%z')
REPO=github.com/dikhan/terraform-provider-openapi
LDFLAGS = '-s -w -extldflags "-static" -X "$(REPO)/openapi/version.Version=$(VERSION)" -X "$(REPO)/openapi/version.Commit=$(COMMIT)" -X "$(REPO)/openapi/version.Date=$(DATE)"'

PROVIDER_NAME?=""
TF_CMD?="plan"

TF_INSTALLED_PLUGINS_PATH="$(HOME)/.terraform.d/plugins"

TEST_PACKAGES?=$$(go list ./... | grep -v "examples\|vendor\|integration")
INT_TEST_PACKAGES?=$$(go list ./... | grep "/tests/integration")
GOFMT_FILES?=$$(find . -name '*.go' | grep -v 'examples\|vendor')

TF_PROVIDER_NAMING_CONVENTION="terraform-provider-"
TF_OPENAPI_PROVIDER_PLUGIN_NAME="$(TF_PROVIDER_NAMING_CONVENTION)openapi"

# By default all are included
DC_SERVICE?=swaggercodegen-service-provider-api swagger-ui-swaggercodegen goa-service-provider-api

default: build

all: test build

# make build
build:
	@echo "[INFO] Building $(TF_OPENAPI_PROVIDER_PLUGIN_NAME) binary"
	@CGO_ENABLED=0 go build -tags=netgo -ldflags=$(LDFLAGS) -o $(TF_OPENAPI_PROVIDER_PLUGIN_NAME)

# make fmt
fmt:
	@echo "[INFO] Running gofmt on the current directory"
	gofmt -s -w $(GOFMT_FILES)

# make vet
vet:
	@echo "[INFO] Running go vet on the current directory"
	@go vet $(TEST_PACKAGES)

# make lint
lint:
	@echo "[INFO] Running golint on the current directory"
	@go get -u golang.org/x/lint/golint
	@golint -set_exit_status $(TEST_PACKAGES)

# make test
test: fmt vet lint
	@echo "[INFO] Testing $(TF_OPENAPI_PROVIDER_PLUGIN_NAME)"
	@go test -v -cover $(TEST_PACKAGES) -coverprofile=coverage.txt -covermode=atomic

# make integration-test
integration-test: local-env-down local-env
	@echo "[INFO] Testing $(TF_OPENAPI_PROVIDER_PLUGIN_NAME)"
	@TF_ACC=true go test -v -cover $(INT_TEST_PACKAGES) ; if [ $$? -eq 1 ]; then \
		echo "[ERROR] Test returned with failures. Please go through the different scenarios and fix the tests that are failing"; \
		exit 1; \
	fi

test-all: test integration-test

pre-requirements:
	@echo "[INFO] Creating $(TF_INSTALLED_PLUGINS_PATH) if it does not exist"
	@[ -d $(TF_INSTALLED_PLUGINS_PATH) ] || mkdir -p $(TF_INSTALLED_PLUGINS_PATH)

release-pre-requirements:
ifeq (, $(shell which github-release-notes))
    @echo "[INFO] No github-release-notes in $(PATH), installing github-release-notes")
    go get github.com/buchanae/github-release-notes
endif
ifeq (, $(shell which goreleaser))
        @echo "[INFO] No goreleaser in $(PATH), installing goreleaser")
        brew install goreleaser
endif

# PROVIDER_NAME="goa" make install
install: build pre-requirements
	$(call install_plugin,$(PROVIDER_NAME))

# make local-env-down
local-env-down: fmt
	@echo "[INFO] Tearing down local environment (clean up task)"
	@docker-compose -f ./build/docker-compose.yml down

# make local-env
local-env: fmt
	@echo "[INFO] Bringing up local environment"
	@docker-compose -f ./build/docker-compose.yml up -d --build --force-recreate $(DC_SERVICE)

# make examples-container
examples-container: local-env
	@echo "[INFO] Bringing up container with OpenAPI providers examples"
	@docker-compose -f ./build/docker-compose.yml build --no-cache terraform-provider-openapi-examples
	@docker-compose -f ./build/docker-compose.yml run terraform-provider-openapi-examples

# [TF_CMD=apply] make run-terraform-example-swaggercodegen
run-terraform-example-swaggercodegen: build pre-requirements
	$(call run_terraform_example,"https://localhost:8443/swagger.yaml",swaggercodegen)

# [TF_CMD=apply] make run-terraform-example-goa
run-terraform-example-goa: build pre-requirements
	$(call run_terraform_example,"http://localhost:9090/swagger/swagger.yaml",goa)

# make latest-tag
latest-tag:
	@echo "[INFO] Latest tag released..."
	@git for-each-ref --sort=-taggerdate --count=1 --format '%(tag)' 'v*' refs/tags

release-notes: release-pre-requirements
	@./scripts/release_notes.sh

# RELEASE_TAG="v0.1.1" GITHUB_TOKEN="PERSONAL_TOKEN" make release-version
release-version: release-notes
	@echo "Attempting to release new version $(CURRENT_RELEASE_TAG); current release $(RELEASE_TAG)"
ifeq ($(NEW_RELEASE_VERSION_VALIDATION),1) # This checks that the new release version present in './version' is greater than the latest version released
	@echo "[INFO] New version $(RELEASE_TAG) valid for release"
	@echo "[INFO] Creating a new tag $(RELEASE_TAG)"
	@git tag -a $(RELEASE_TAG) -m $(RELEASE_TAG)
	@echo "[INFO] Releasing $(RELEASE_TAG)"
	@GITHUB_TOKEN=$(GITHUB_TOKEN) goreleaser --rm-dist --release-notes ./release-notes.md
else
	@echo "Cancelling release due to new version $(RELEASE_TAG) <= latest release version $(CURRENT_RELEASE_TAG)"
endif

define install_plugin
    @$(eval TF_PROVIDER_PLUGIN_NAME := $(TF_PROVIDER_NAMING_CONVENTION)$(1))

	@echo "[INFO] Installing $(TF_PROVIDER_PLUGIN_NAME) binary in -> $(TF_INSTALLED_PLUGINS_PATH)/$(TF_PROVIDER_PLUGIN_NAME)"
	@mv ./$(TF_OPENAPI_PROVIDER_PLUGIN_NAME) $(TF_INSTALLED_PLUGINS_PATH)/$(TF_PROVIDER_PLUGIN_NAME)
endef

define run_terraform_example
    @$(eval OTF_VAR_SWAGGER_URL := $(1))
    @$(eval PROVIDER_NAME := $(2))

	$(call install_plugin,$(PROVIDER_NAME))

    @$(eval TF_EXAMPLE_FOLDER := ./examples/$(PROVIDER_NAME))

	@echo "[INFO] Performing sanity check against the service provider's swagger endpoint '$(OTF_VAR_SWAGGER_URL)'"
	@$(eval SWAGGER_HTTP_STATUS := $(shell curl -s -o /dev/null -w '%{http_code}' $(OTF_VAR_SWAGGER_URL) -k))
	@if [ "$(SWAGGER_HTTP_STATUS)" = 200 ]; then\
        echo "[INFO] Terraform Configuration file located at $(TF_EXAMPLE_FOLDER)";\
        echo "[INFO] Executing TF command: OTF_INSECURE_SKIP_VERIFY=true OTF_VAR_$(PROVIDER_NAME)_SWAGGER_URL=$(OTF_VAR_SWAGGER_URL) && terraform init && terraform ${TF_CMD}";\
        cd $(TF_EXAMPLE_FOLDER) && export OTF_INSECURE_SKIP_VERIFY=true OTF_VAR_$(PROVIDER_NAME)_SWAGGER_URL=$(OTF_VAR_SWAGGER_URL) && terraform init && terraform ${TF_CMD};\
    else\
        echo "[ERROR] Sanity check against swagger endpoint[$(OTF_VAR_SWAGGER_URL)] failed...Please make sure the service provider API is up and running and exposes swagger APIs on '$(OTF_VAR_SWAGGER_URL)'";\
    fi
endef

.PHONY: all build fmt vet lint test run_terraform
