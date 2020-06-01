#!/bin/bash

BIN="$(cd "$(dirname "$0")" ; pwd)"
SRC="$(dirname "${BIN}")"
PROJECT="$(dirname "${SRC}")"

source "${BIN}/verbose.sh"
source "${SRC}/etc/settings-local.sh"

function run-with-protoc() {
  if type protoc >/dev/null 2>&1
  then
    (
      cd "${BIN}"
      "$@"
    )
  else
    docker run --rm -v "${PROJECT}:${PROJECT}" -w "${BIN}" "${DOCKER_REPOSITORY}/build-protoc" "$@"
  fi
}

(
  run-with-protoc ./generate-proto-go-package.sh -v
  cd "${PROJECT}"
  "${BIN}/nix.sh" nix-build -E 'with import <nixpkgs> {}; pkgs.callPackage ./default.nix {}'
)
