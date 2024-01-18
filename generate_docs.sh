#!/bin/bash

PROTOS_DIR="protos"
DOCS_DIR="docs"

# Expected structure of protos dir for document generation.
#
#  protos
#  |- entity
#       |- v1
#           |- entity.proto

set -e

rm -rf $DOCS_DIR
mkdir -p $DOCS_DIR


entity_dirs=`find $PROTOS_DIR -maxdepth 1 -mindepth 1 -type d`
for entity_dir in $entity_dirs; do
    entity=`basename $entity_dir`
    version_dirs=`find $entity_dir -maxdepth 1 -mindepth 1 -type d`
    for version_dir in $version_dirs; do
        version=`basename $version_dir`
        proto_files=`find $version_dir -name *.proto -type f`
        protoc --proto_path=${PROTOS_DIR} \
            --doc_out=${DOCS_DIR} \
            --doc_opt=markdown,${entity}_${version}.md \
            ${proto_files}
    done
done

set +e
exit 0
