include ./e2/builder/builder.mk
include ./e2/cli/release/release.mk

GO_INSTALL=go install -ldflags $(RELEASE_FLAGS) ./e2

e2:
	$(GO_INSTALL)

e2/dev:
	$(GO_INSTALL) -tags=development

e2/docker-bin:
	$(GO_INSTALL) -tags=docker

e2/docker:
	DOCKER_BUILDKIT=1 docker build ./e2 -t suborbital/e2:dev

e2/docker/publish:
	docker buildx build ./e2 --platform linux/amd64,linux/arm64 -t suborbital/e2:dev --push

e2/smoketest: e2
	./e2/scripts/smoketest.sh


.PHONY: e2 e2/dev e2/docker-bin e2/docker e2/docker/publish e2/smoketest
