#!/bin/bash

set -e

BIN="$(cd "$(dirname "$0")" ; pwd)"
SRC="$(dirname "${BIN}")"
PROJECT="$(dirname "${SRC}")"

source "${BIN}/verbose.sh"

AXON_PROTO="${PROJECT}/data/axon-server-api/src/main/proto"

if [[ -d "${AXON_PROTO}" ]]
then
  cd "${AXON_PROTO}"
  log "Generating Go stubs from $(pwd)"

  sed -E -i \
    -e 's/^option/old_option/' \
    -e "3i\\
option go_package = \"src/pkg/grpc/axon_server\";" \
    -e '/^old_option go_package =/d' \
    -e 's/^old_option/option/' \
    *.proto

  protoc --go_out="plugins=grpc:${PROJECT}" -I. *.proto
fi

cd "${SRC}/proto"
log "Generating Go stubs from $(pwd)"
protoc --go_out="plugins=grpc:${PROJECT}" -I. *.proto
