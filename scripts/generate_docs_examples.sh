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

EXAMPLES_DIR="${DOCS_DIR}/examples"

echo "Clearing examples directory -> $EXAMPLES_DIR"
rm -rf $EXAMPLES_DIR
mkdir -p $EXAMPLES_DIR

echo "Generating examples..."
index_content="""
## Examples

You can find some language specific examples in the linked pages.
"""

for folder in $EXAMPLES_FOLDER/*/; do
    lang=$(basename "${folder}")
    lang=${lang//[ ]/_}
    extension=""
    codelang=""

    if [[ "${lang}" == "golang" ]]; then
        extension="go"
        codelang="go"
    elif [[ "${lang}" == "python" ]]; then
        extension="py"
        codelang="python"
    else
        continue
    fi

    echo "** folder='${folder}', lang='${lang}', ext='${extension}'"

    index_content="""${index_content}

#### ${lang}
"""

    for example_folder in $folder*/; do
        readarray -d '' code_files < <(find $example_folder -type f -name \*.$extension -print0)
        if [[ "${#code_files[@]}" == "0" ]]; then
            continue
        fi

        example_name=$(basename $example_folder)

        index_content="""${index_content}
* [${example_name}](examples/${lang}/${example_name})
"""
        
        content="""
### ${example_name}

Find full example [here](https://github.com/getsynq/api/tree/main/examples/${lang}/${example_name})
"""

        IFS=$'\n' sorted=($(sort <<<"${code_files[*]}")); unset IFS
        for code_file in "${sorted[@]}"; do
            filename_with_ext=$(basename -- "$code_file")
            file_name="${code_file%.*}"
            code=$(<$code_file)

            content="""${content}

#### ${filename_with_ext}
\`\`\`${codelang}
${code}
\`\`\`
"""
        done

        # write content out to file
        dest_file="${EXAMPLES_DIR}/${lang}/${example_name}.mdx"
        mkdir -p $(dirname "${dest_file}")
        echo "$content" > "$dest_file"
    done
done

dest_file="${DOCS_DIR}/examples.mdx"
echo "$index_content" > "$dest_file"

set +e
exit 0
