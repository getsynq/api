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

INCLUDES=()
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
        INCLUDES+=("$2")
        shift 2
        ;;
    --) shift; 
        break 
        ;;
  esac
done

set -e

echo "Generating docs..."

source "${MY_PATH}/buf_workspace.sh"
buf_workspace "${PROTOS_DIR}" "${INCLUDES[@]}"

# Every .proto file under the protos directory, in one request, so the template renders
# a single page.
mkdir -p "${DOCS_DIR}"
DOCS_DIR=$(cd "${DOCS_DIR}" && pwd)
(cd "${BUF_WORKSPACE}" && buf generate protos --template "
version: v2
plugins:
  - local: protoc-gen-doc
    out: ${DOCS_DIR}
    opt: ${TEMPLATE},api.mdx
    strategy: all
")

echo "Escaping MDX special characters..."
python3 "${MY_PATH}/escape_mdx.py" "${DOCS_DIR}/api.mdx"

set +e
exit 0
