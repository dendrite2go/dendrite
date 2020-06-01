#!/bin/bash

BIN="$(cd "$(dirname "$0")" ; pwd)"
SRC="$(dirname "${BIN}")"
PROJECT="$(dirname "${SRC}")"

source "${BIN}/verbose.sh"
source "${SRC}/etc/settings-local.sh"

BIND="${PROJECT}"
if [ ".$1" = '.--bind' ]
then
  BIND="$2"
  shift 2
fi

mkdir -p ~/.cache/nix

if type nix >/dev/null 2>&1
then
  "$@"
else
  docker run -ti --rm -v "${NIX_STORE_VOLUME}:/nix/store" -v "${HOME}/.cache/nix:/root/.cache/nix" -v "${BIND}:${BIND}" -w "$(pwd)" "${DOCKER_REPOSITORY}/nix-go-protobuf" "$@"
fi