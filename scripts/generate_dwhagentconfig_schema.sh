#!/bin/bash

MY_PATH="$(dirname -- "${BASH_SOURCE[0]}")"
TEMPLATE=$(realpath "$MY_PATH/../templates/grpc-md.tmpl")

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


echo "Generating DWH agent config docs..."
protoc  --proto_path="${PROTOS_DIR}" ${PROTOC_OPTS} --jsonschema_opt=enforce_oneof '--jsonschema_opt=messages=[Config]' --jsonschema_opt=enums_as_strings_only --jsonschema_out=${DOCS_DIR} ${PROTOS_DIR}/synq/agent/dwh/v1/dwh_agent_config.proto
mv ${DOCS_DIR}/Config.json ${DOCS_DIR}/dwhagentconfig.schema.json

cat ${DOCS_DIR}/dwhagentconfig.schema.json  | docker run --rm -i elisiariocouto/jsonschema-markdown --no-empty-columns --examples-format yaml - | sed 's/synq.agent.dwh.v1.//g' | sed 's/#config./#config-/g' > ${DOCS_DIR}/dwhagentconfig.schema.md

set +e
exit 0
