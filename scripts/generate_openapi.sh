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

INCLUDES=("${PROTOS_DIR}/../proto_shared")
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
  - url: https://api.us.synq.io
  - url: https://api.au.synq.io
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
EOF

source "${MY_PATH}/buf_workspace.sh"
buf_workspace "${PROTOS_DIR}" "${INCLUDES[@]}"

# The inputs are the files declaring an HTTP route, given to the plugin in one request so
# it writes a single spec.
targets=()
for f in $(cd "${PROTOS_DIR}" && grep -r -l "google.api.http" synq); do
    targets+=(--path "protos/${f}")
done
# Without a --path, buf would generate for every file in the workspace, includes too.
if [[ ${#targets[@]} -eq 0 ]]; then
    echo "no proto file under ${PROTOS_DIR}/synq declares google.api.http" >&2
    exit 1
fi

echo "Generating swagger file..."
mkdir -p "${DOCS_DIR}"
DOCS_DIR=$(cd "${DOCS_DIR}" && pwd)
(cd "${BUF_WORKSPACE}" && buf generate "${targets[@]}" --template "
version: v2
plugins:
  - local: protoc-gen-connect-openapi
    out: ${DOCS_DIR}
    strategy: all
    opt:
      - base=${basefile}
      - path=openapi.yaml
      - content-types=json
      - only-googleapi-http
      - trim-unused-types
")

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
