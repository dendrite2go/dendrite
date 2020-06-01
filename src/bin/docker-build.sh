#!/bin/bash

BIN="$(cd "$(dirname "$0")" ; pwd)"
SRC="$(dirname "${BIN}")"
DOCKER="${SRC}/docker"

source "${SRC}/etc/settings-local.sh"

docker build -t "${DOCKER_REPOSITORY}/nix-go-protobuf" "${DOCKER}"
docker build -t "${DOCKER_REPOSITORY}/build-protoc" -f "${DOCKER}/Dockerfile-protoc" "${DOCKER}"