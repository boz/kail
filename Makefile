
DOCKER_IMAGE ?= kail
DOCKER_REPO  ?= abozanich/$(DOCKER_IMAGE)
DOCKER_TAG   ?= latest

IMG_LDFLAGS := -w -linkmode external -extldflags "-static"

build:
	govendor build -i +program

ifeq ($(shell uname -s),Darwin)
build-linux:
	GOOS=linux GOARCH=amd64 go build -o kail-linux ./cmd/kail
else
build-linux:
	CC=$$(which musl-gcc) go build --ldflags '$(IMG_LDFLAGS)' -o kail-linux ./cmd/kail
endif

test:
	govendor test +local

test-full: build
	govendor test -v -race +local

image: build-linux
	docker build -t $(DOCKER_IMAGE) .

image-minikube: build-linux
	eval $$(minikube docker-env) && docker build -t $(DOCKER_IMAGE) .

image-push: image
	docker tag $(DOCKER_IMAGE) $(DOCKER_REPO):$(DOCKER_TAG)
	docker push $(DOCKER_REPO):$(DOCKER_TAG)

install-libs:
	govendor install +vendor,^program

install-deps:
	go get github.com/kardianos/govendor
	govendor sync

release:
	GITHUB_TOKEN=$$GITHUB_REPO_TOKEN goreleaser

clean:
	rm kail kail-linux 2>/dev/null || true

.PHONY: build build-linux \
	test test-full \
	image image-minikube image-push \
	install-libs install-deps \
	clean
