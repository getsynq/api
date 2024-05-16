#!/bin/bash

PROTOS_DIR="."

set -e

VALID_ARGS=$(getopt -o I: -- "$@")
if [[ $? -ne 0 ]]; then
    exit 1;
fi

PROTOC_OPTS=""
eval set -- "$VALID_ARGS"
while [ : ]; do
  case "$1" in
    -I)
        PROTOC_OPTS="$PROTOC_OPTS --proto_path=$2"
        shift 2
        ;;
    --) shift;
        break
        ;;
  esac
done

echo "Generating docs..."

# Find all .proto files in subdirectories
proto_files=$(find "${PROTOS_DIR}" -type f -name "*.proto")

protoc --proto_path="${PROTOS_DIR}" ${PROTOC_OPTS}\
    --doc_out=_repo/templates/grpc-md.tmpl,api.mdx:. \
    ${proto_files}

mv api.mdx ../../docs/api-reference/api.mdx

set +e
exit 0