#!/bin/bash

PROTOS_DIR=""
GEN_DIR="./tmp"
DIRTY=false

set -e

VALID_ARGS=$(getopt -o dp:g: --long dirty,protos:,gen: -- "$@")
if [[ $? -ne 0 ]]; then
    exit 1;
fi

eval set -- "$VALID_ARGS"
while [ : ]; do
  case "$1" in
    -d | --dirty)
        DIRTY=true
        shift
        ;;
    -p | --protos)
        PROTOS_DIR=$2
        shift 2
        ;;
    -g | --gen)
        GEN_DIR=$2
        shift 2
        ;;
    --) shift; 
        break 
        ;;
  esac
done

echo "Clearing generated directory -> $GEN_DIR"
rm -rf $GEN_DIR
mkdir -p $GEN_DIR
DOCS_DIR="$GEN_DIR/_docs"
mkdir -p $DOCS_DIR

echo "Compiling protos from -> $PROTOS_DIR"
entity_dirs=`find $PROTOS_DIR -maxdepth 1 -mindepth 1 -type d`
for entity_dir in $entity_dirs; do
    echo "processing $entity_dir"
    entity=`basename $entity_dir`
    mkdir "${DOCS_DIR}/${entity}"
    version_dirs=`find $entity_dir -maxdepth 1 -mindepth 1 -type d`
    for version_dir in $version_dirs; do
        version=`basename $version_dir`
        proto_files=`find $version_dir -name *.proto -type f`
        if [[ "${proto_files}" != "" ]]; then
            protoc --proto_path=${PROTOS_DIR} \
                --go_out=${GEN_DIR} \
                --go-grpc_out=${GEN_DIR} \
                --doc_out=${DOCS_DIR}/${entity} \
                --doc_opt=markdown,${version}.md \
                ${proto_files}
        fi
    done
    if [ -z "$(ls -A ${DOCS_DIR}/${entity})" ]; then
        rm -r "${DOCS_DIR}/${entity}"
    fi
done

if [ "$DIRTY" = false ] ; then
    echo 'Removing generated directory.'
    rm -rf $GEN_DIR
fi

set +e
exit 0