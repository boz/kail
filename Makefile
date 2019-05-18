
DOCKER_IMAGE ?= kail
DOCKER_REPO  ?= abozanich/$(DOCKER_IMAGE)
DOCKER_TAG   ?= latest

BUILD_ENV = GOOS=linux GOARCH=amd64

GO = GO111MODULE=on go

ifdef TRAVIS
	LDFLAGS += -X main.version=$(TRAVIS_BRANCH) -X main.commit=$(TRAVIS_COMMIT)
endif

build:
	$(GO) build -o kail ./cmd/kail

build-linux:
	$(BUILD_ENV) $(GO) build --ldflags '$(LDFLAGS)'  -o kail-linux ./cmd/kail

test:
	$(GO) test ./...

test-full: build image
	$(GO) test -v -race ./...

image: build-linux
	docker build -t $(DOCKER_IMAGE) .

image-minikube: build-linux
	eval $$(minikube docker-env) && docker build -t $(DOCKER_IMAGE) .

image-push: image
	docker tag $(DOCKER_IMAGE) $(DOCKER_REPO):$(DOCKER_TAG)
	docker push $(DOCKER_REPO):$(DOCKER_TAG)

install-deps:
	$(GO) mod download

release:
	GITHUB_TOKEN=$$GITHUB_REPO_TOKEN goreleaser -f .goreleaser.yml

clean:
	rm kail kail-linux dist 2>/dev/null || true

.PHONY: build build-linux \
	test test-full \
	image image-minikube image-push \
	install-deps \
	clean
