#!/bin/bash

PROTOS_DIR="."
DOCS_DIR="./tmp"

set -e

VALID_ARGS=$(getopt -o p:d: --long protos:,docs: -- "$@")
if [[ $? -ne 0 ]]; then
    exit 1;
fi

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
    --) shift; 
        break 
        ;;
  esac
done

set -e

echo "Clearing docs directory -> $DOCS_DIR"
rm -rf $DOCS_DIR
mkdir -p $DOCS_DIR

echo "Generating docs..."
module_dirs=`find $PROTOS_DIR -maxdepth 1 -mindepth 1 -type d`
for module_dir in $module_dirs; do
    echo "processing $module_dir"
    module=`basename $module_dir`
    mkdir "${DOCS_DIR}/${module}"
    version_dirs=`find $module_dir -maxdepth 1 -mindepth 1 -type d`
    for version_dir in $version_dirs; do
        version=`basename $version_dir`
        proto_files=`find $version_dir -name *.proto -type f`
        if [[ "${proto_files}" != "" ]]; then
            protoc --proto_path=${PROTOS_DIR} \
                --doc_out="${DOCS_DIR}/${module}/" \
                --doc_opt=markdown,${version}.md \
                ${proto_files}
        fi
    done
    if [ -z "$(ls -A ${DOCS_DIR}/${module})" ]; then
        rm -r "${DOCS_DIR}/${module}"
    fi
done

set +e
exit 0
