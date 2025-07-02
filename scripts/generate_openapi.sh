#!/bin/bash

MY_PATH="$(dirname -- "${BASH_SOURCE[0]}")"

PROTOS_DIR="${MY_PATH}/../.."
DOCS_DIR="${MY_PATH}/.."

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

PROTOC_OPTS="--proto_path=${PROTOS_DIR}/../proto_shared"
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


echo "Generating swagger files..."
protoc --proto_path="${PROTOS_DIR}" ${PROTOC_OPTS} --openapi_out=${DOCS_DIR} --openapi_opt='title=SYNQ,version=v1,description=REST API interface for SYNQ' $(grep -r -l "google.api.http" ${PROTOS_DIR}/synq)

yq -i -I 4 -e -p yaml -o yaml '
    .security[0].bearerAuth = [] |
    .components.securitySchemes.bearerAuth.type = "http" |
    .components.securitySchemes.bearerAuth.scheme = "bearer" |
    .servers[0].url = "https://developer.synq.io"
' "${DOCS_DIR}"/openapi.yaml

set +e
exit 0
