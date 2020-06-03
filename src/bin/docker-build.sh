#!/bin/bash

BIN="$(cd "$(dirname "$0")" ; pwd)"
SRC="$(dirname "${BIN}")"

source "${BIN}/verbose.sh"
source "${SRC}/etc/settings-local.sh"

DOCKER="${SRC}/docker"
PROJECT="$(dirname "${SRC}")"
TARGET="${PROJECT}/target"

docker build -t "${DOCKER_REPOSITORY}/nix-go-protobuf" "${DOCKER}/nix-go-protobuf"
docker build -t "${DOCKER_REPOSITORY}/build-protoc" "${DOCKER}/build-protoc"

function interpolate() {
  BINARY="$1"
  sed -e "s:[\$]{BINARY}:${BINARY}:g" "${DOCKER}/binary/Dockerfile" > "${TARGET}/Dockerfile-${BINARY}"
}

for B in "${TARGET}/bin"/*
do
  BINARY="$(basename "${B}")"
  info "BINARY: [${BINARY}]"
  interpolate "${BINARY}"
  docker build -t "${DOCKER_REPOSITORY}/${BINARY}" -f "${TARGET}/Dockerfile-${BINARY}" "${TARGET}"
done
