#!/bin/bash

MY_PATH="$(dirname -- "${BASH_SOURCE[0]}")"
TEMPLATE="$MY_PATH/../templates/grpc-md.tmpl"

PROTOS_DIR="."
DOCS_DIR="./tmp"

set -e

VALID_ARGS=$(getopt -o p:d:I: --long protos:,docs: -- "$@")
if [[ $? -ne 0 ]]; then
    exit 1;
fi

PROTOC_OPTS=""
eval set -- "$VALID_ARGS"
while [ : ]; do
  case "$1" in
    -p | --protos)
        PROTOS_DIR=$2
        shift 2
        ;;
    -d | --docs)
        DOCS_DIR=$2
        shift 2
        ;;
    -I)
        PROTOC_OPTS="$PROTOC_OPTS --proto_path=$2"
        shift 2
        ;;
    --) shift; 
        break 
        ;;
  esac
done

set -e

echo "Generating docs..."

# Find all .proto files in subdirectories
proto_files=$(find "${PROTOS_DIR}" -type f -name "*.proto")

protoc --proto_path="${PROTOS_DIR}" ${PROTOC_OPTS}\
    --doc_out=${TEMPLATE},${DOCS_DIR}/${module}/api.mdx:. \
    ${proto_files}

set +e
exit 0
