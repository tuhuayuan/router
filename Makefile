include includes.mk

SHORT_NAME := router
DEIS_REGISTRY ?= ${DEV_REGISTRY}
IMAGE_PREFIX ?= deis

include versioning.mk

SHELL_SCRIPTS = $(wildcard _scripts/*.sh) $(wildcard rootfs/bin/*) rootfs/opt/router/sbin/boot

REPO_PATH := github.com/deis/${SHORT_NAME}

# The following variables describe the containerized development environment
# and other build options
DEV_ENV_IMAGE := index.tenxcloud.com/tuhuayuan/go-dev:v1.0
DEV_ENV_WORK_DIR := /go/src/${REPO_PATH}
DEV_ENV_CMD := docker run --rm -v ${CURDIR}:${DEV_ENV_WORK_DIR} -w ${DEV_ENV_WORK_DIR} ${DEV_ENV_IMAGE}
DEV_ENV_CMD_INT := docker run -it --rm -v ${CURDIR}:${DEV_ENV_WORK_DIR} -w ${DEV_ENV_WORK_DIR} ${DEV_ENV_IMAGE}
LDFLAGS := "-s -w -X main.version=${VERSION}"
BINDIR := ./rootfs/opt/router/sbin

# The following variables describe the source we build from
GO_FILES := $(wildcard *.go)
GO_DIRS := model/ nginx/ utils/ utils/modeler
GO_PACKAGES := ${REPO_PATH} $(addprefix ${REPO_PATH}/,${GO_DIRS})

# The binary compression command used
UPX := upx -9 --mono --no-progress

# The following variables describe k8s manifests we may wish to deploy
# to a running k8s cluster in the course of development.
RC := manifests/deis-${SHORT_NAME}-rc.yaml
SVC := manifests/deis-${SHORT_NAME}-service.yaml

# Allow developers to step into the containerized development environment
dev: check-docker
	${DEV_ENV_CMD_INT} bash

dev-registry: check-docker
	@docker inspect registry >/dev/null 2>&1 && docker start registry || docker run --restart="always" -d -p 5000:5000 --name registry index.tenxcloud.com/docker_library/registry:2
	@echo
	@echo "To use a local registry for Deis development:"
	@echo "    export DEIS_REGISTRY=`docker-machine ip $$(docker-machine active 2>/dev/null) 2>/dev/null || echo DOCKER_HOST `:5000/"

# Containerized dependency resolution
bootstrap: check-docker
	${DEV_ENV_CMD} glide install

# Containerized build of the binary
build: check-docker
	mkdir -p ${BINDIR}
	${DEV_ENV_CMD} make binary-build

docker-build: build check-docker
	docker build --rm -t ${IMAGE} rootfs
	docker tag ${IMAGE} ${MUTABLE_IMAGE}

# Builds the binary-- this should only be executed within the
# containerized development environment.
binary-build:
	GOOS=linux GOARCH=amd64 go build -o ${BINDIR}/${SHORT_NAME} -ldflags ${LDFLAGS} ${SHORT_NAME}.go
	$(call check-static-binary,$(BINDIR)/${SHORT_NAME})
	${UPX} ${BINDIR}/${SHORT_NAME}

clean: check-docker
	docker rmi ${IMAGE}

full-clean: check-docker
	docker images -q ${DEIS_REGISTRY}/${IMAGE_PREFIX}/${SHORT_NAME} | xargs docker rmi -f

dev-release: docker-build docker-push set-image

set-image:
	sed "s#\(image:\) .*#\1 ${IMAGE}#" manifests/deis-${SHORT_NAME}-rc.yaml > manifests/deis-${SHORT_NAME}-rc.tmp.yaml
	sed "s#\(image:\) .*#\1 ${IMAGE}#" manifests/deis-${SHORT_NAME}-ds.yaml > manifests/deis-${SHORT_NAME}-ds.tmp.yaml

deploy-rc: check-kubectl dev-release
	@kubectl describe rc deis-${SHORT_NAME} --namespace=kube-system >/dev/null 2>&1; \
	if [ $$? -eq 0 ]; then \
		kubectl delete rc deis-${SHORT_NAME} --namespace=kube-system; \
		kubectl create -f manifests/deis-${SHORT_NAME}-rc.tmp.yaml; \
	else \
		kubectl create -f manifests/deis-${SHORT_NAME}-rc.tmp.yaml; \
	fi

deploy-ds: check-kubectl dev-release
	@kubectl describe daemonsets deis-${SHORT_NAME} --namespace=kube-system >/dev/null 2>&1; \
	if [ $$? -eq 0 ]; then \
		kubectl delete daemonsets deis-${SHORT_NAME} --namespace=kube-system; \
		kubectl create -f manifests/deis-${SHORT_NAME}-ds.tmp.yaml; \
	else \
		kubectl create -f manifests/deis-${SHORT_NAME}-ds.tmp.yaml; \
	fi

examples:
	kubectl create -f manifests/examples.yaml

test: test-style test-unit test-functional

test-cover:
	${DEV_ENV_CMD} test-cover.sh

test-functional:
	@echo no functional tests

test-style: check-docker
	${DEV_ENV_CMD} make style-check

# This should only be executed within the containerized development environment.
style-check:
	lint
	shellcheck $(SHELL_SCRIPTS)

test-unit:
	${DEV_ENV_CMD} go test --cover --race -v ${GO_PACKAGES}
