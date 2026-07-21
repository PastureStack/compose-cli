TARGETS := $(shell ls scripts)
DAPPER_IMAGE ?= pasturestack-compose-cli-dapper:ubuntu26
DAPPER_SOURCE ?= /go/src/github.com/PastureStack/compose-cli
DOCKER_VERSION ?= 29.4.2

.dapper-image: Dockerfile.dapper
	docker build \
		--build-arg DAPPER_HOST_ARCH=$${DAPPER_HOST_ARCH:-amd64} \
		--build-arg DOCKER_VERSION=$(DOCKER_VERSION) \
		-t $(DAPPER_IMAGE) \
		-f Dockerfile.dapper .

$(TARGETS): .dapper-image
	docker run --rm \
		-v $(CURDIR):$(DAPPER_SOURCE) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-e DAPPER_UID=$$(id -u) \
		-e DAPPER_GID=$$(id -g) \
		-e TAG \
		-e REPO \
		-e CROSS \
		-e VERSION_OVERRIDE \
		-e SKIP_INTEGRATION \
		-e PLATFORM_COMPAT_JAR_URL \
		-e CATTLE_JAR_URL \
		$(DAPPER_IMAGE) $@

deps:
	@echo "Dependencies are vendored; no external dependency fetch is required."

.DEFAULT_GOAL := ci

.PHONY: $(TARGETS) deps
