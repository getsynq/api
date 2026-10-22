#!/bin/bash
#
# Sourced by the scripts beside it, which compile the protos with buf rather than protoc.
#
# buf resolves imports through a workspace of modules, where protoc takes a list of
# --proto_path roots. buf_workspace turns the second into the first: it creates a temporary
# directory holding one symlink per root (the protos directory first, then every -I
# directory) and a buf.yaml naming each as a module. The roots' own buf.yaml files, if
# any, are not read, so imports resolve exactly as they do with protoc and nothing is
# fetched from the Buf Schema Registry.
#
#   buf_workspace <protos dir> [<include dir>...]
#   buf generate "${BUF_WORKSPACE}" --path "${BUF_WORKSPACE}/protos/<path inside the protos dir>"
#
# It sets BUF_WORKSPACE rather than printing it, because the cleanup trap it installs has
# to belong to the calling script: inside $(...) it would fire, and remove the workspace,
# as soon as the substitution returned.

buf_workspace() {
    if ! command -v buf >/dev/null 2>&1; then
        echo "buf is required, see https://buf.build/docs/cli/installation/" >&2
        exit 1
    fi

    BUF_WORKSPACE=$(mktemp -d)
    # The workspace holds only symlinks, so removing it never touches the protos.
    trap 'rm -rf "${BUF_WORKSPACE}"' EXIT

    local config="version: v2
modules:"
    local name=protos
    local i=0
    local seen=" "
    local dir
    local abs
    for dir in "$@"; do
        # protoc only warns about a missing --proto_path, so a missing include is skipped
        # the same way. The protos directory itself is always required.
        if [[ ! -d "${dir}" ]]; then
            if [[ "${name}" == protos ]]; then
                echo "protos directory not found: ${dir}" >&2
                exit 1
            fi
            echo "warning: include directory not found, skipping: ${dir}" >&2
            continue
        fi
        abs=$(cd "${dir}" && pwd -P)
        # protoc tolerates the same root given twice; two buf modules holding the same
        # files are a build error.
        if [[ "${seen}" == *" ${abs} "* ]]; then
            continue
        fi
        seen="${seen}${abs} "
        ln -s "${abs}" "${BUF_WORKSPACE}/${name}"
        config="${config}
  - path: ${name}"
        i=$((i + 1))
        name="include${i}"
    done
    echo "${config}" >"${BUF_WORKSPACE}/buf.yaml"
}
