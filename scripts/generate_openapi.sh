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

base=$(mktemp -d)
basefile="$base/base.openapi.yaml"
echo $basefile
tee "$basefile" <<EOF
openapi: 3.1.0
info:
  version: "1.0"
  title: "SYNQ"
security:
  - bearerAuth: []
servers:
  - url: https://developer.synq.io
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
EOF

echo "Generating swagger file..."
protoc \
    --proto_path="${PROTOS_DIR}" \
    ${PROTOC_OPTS} \
    --connect-openapi_out=${DOCS_DIR} \
    --connect-openapi_opt=base=$basefile \
    --connect-openapi_opt=path=openapi.yaml \
    --connect-openapi_opt=content-types=json \
    $(grep -r -l "google.api.http" ${PROTOS_DIR}/synq)

set +e
exit 0
