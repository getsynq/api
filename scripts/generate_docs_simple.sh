#!/bin/bash

MY_PATH="$(dirname -- "${BASH_SOURCE[0]}")"
TEMPLATE=$(realpath "$MY_PATH/../templates/grpc-md.tmpl")

PROTOS_DIR="${MY_PATH}/../../"
DOCS_DIR="./tmp"

set -e

GETOPT=$(which getopt)
if [[ -x "/opt/homebrew/opt/gnu-getopt/bin/getopt" ]]
then
    GETOPT="/opt/homebrew/opt/gnu-getopt/bin/getopt"
fi

VALID_ARGS=$(${GETOPT} -o p:d:I: --long protos:,docs: -- "$@")
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
    --doc_out="${TEMPLATE},api.mdx:${DOCS_DIR}" \
    ${proto_files}

echo "Escaping MDX special characters..."
python3 "${MY_PATH}/escape_mdx.py" "${DOCS_DIR}/api.mdx"

set +e
exit 0
