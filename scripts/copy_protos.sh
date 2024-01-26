#!/bin/bash

if [ "$#" -ne 2 ]; then
    echo "Need source and destination"
fi
source=$1
destination=$2

set -e

proto_files=`find $source -name *.proto -type f`
declare -a proto_dirs

# find unique proto directories in source
for pf in $proto_files; do
    pd=`dirname "${pf}"`
    if ! [[ $(echo ${proto_dirs[@]} | fgrep -w $pd) ]]; then
        proto_dirs+=("$pd")
    fi
done

# copy proto directories to same file path in destination
for pd in "${proto_dirs[@]}"; do
    dest="${pd/$source/$destination}"
    # ensure dest exists
    mkdir -p $dest
    echo "$pd -> $dest"
    rsync -avW --include='*.proto' --no-compress $pd $dest --delete
done

set +e
exit 0
