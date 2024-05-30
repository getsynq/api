#!/bin/bash

MY_PATH="$(dirname -- "${BASH_SOURCE[0]}")"
EXAMPLES_FOLDER="${MY_PATH}/../examples"
DOCS_DIR="./tmp"

set -e

VALID_ARGS=$(getopt -o f:d: --long folder:,docs: -- "$@")
if [[ $? -ne 0 ]]; then
    exit 1;
fi

eval set -- "$VALID_ARGS"
while [ : ]; do
  case "$1" in
    -f | --folder)
        EXAMPLES_FOLDER=$2
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

echo "Generating examples..."
for folder in $EXAMPLES_FOLDER/*/; do
    lang=$(basename "${folder}")
    lang=${lang//[ ]/_}
    extension=""

    if [[ "${lang}" == "golang" ]]; then
        extension="go"
    else
        continue
    fi

    echo "** folder='${folder}', lang='${lang}', ext='${extension}'"

    for example_folder in $folder*/; do
        readarray -d '' code_files < <(find $example_folder -type f -name \*.$extension -print0)
        if [[ "${#code_files[@]}" == "0" ]]; then
            continue
        fi

        example_name=$(basename $example_folder)
        
        content="""
### ${example_name}

Find full example [here](https://github.com/getsynq/api/examples/${lang}/${example_name})
"""

        for code_file in "${code_files[@]}"; do
            filename_with_ext=$(basename -- "$code_file")
            file_name="${code_file%.*}"
            code=$(<$code_file)

            content="""${content}

#### ${filename_with_ext}
\`\`\`
${code}
\`\`\`
"""
        done

        # write content out to file
        dest_file="${DOCS_DIR}/${lang}/${example_name}.mdx"
        mkdir -p $(dirname "${dest_file}")
        echo "$content" > "$dest_file"
    done
done

set +e
exit 0
