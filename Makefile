# re-connect.ai container image
REGISTRY_USERNAME ?= todo
IMAGE_NAME ?= reconn
IMAGE_TAG ?= latest

.PHONY: all
all:
	cd reconn-webapp && ng build --base-href /resource/ && cd ..
	go build
	docker build --tag ${REGISTRY_USERNAME}/${IMAGE_NAME}:${IMAGE_TAG} .
