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

# Having an explicit title on any of the schemas that use `oneOf` makes the mintlify docs hard to read
# since it uses that title for each variant, even though the variants all have explicit titles as well.
# As a workaround for that, we remove the title property from any schema object that has `oneOf`.
#
# Relevant issue: https://linear.app/synq/issue/PR-5531/mintlify-docs-renders-oneof-variants-poorly-each-variant-has-the-same
yq \
  --inplace \
  --indent=4 \
  --input-format=yaml \
  --output-format=yaml \
  '
    del(.components.schemas.[] | select (.oneOf != null and .title != null).title) |
    
    # Similarly, we delete the title from any properties that use a reference,
    # as that also overrides the title used by Mintlify.
    del(.components.schemas.[] | select (.properties != null).properties.[] | select (".$ref" != null and .title != null).title) |

    # Lastly, we need to do the same for any paths that specify these titles explicitly
    del(.. | select(has("schema")).schema | select(has("properties")).properties.[] | select(has("$ref") and has("title")).title)
    ' \
  "${DOCS_DIR}/openapi.yaml"


set +e
exit 0
