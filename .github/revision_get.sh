#!/bin/bash

export SCRIPT_DIR=$(dirname $(readlink -f $0))
export PROJECT_DIR=$(dirname $SCRIPT_DIR)

main() {
    major_minor=$(cat $PROJECT_DIR/VERSION)
    revision=$(revision_get $major_minor)
    echo "$major_minor.$revision"
}

#
# Usage:
#
# revision_get 0.0
#   0
revision_get() {
    major_mino=$1
    key="rev-$major_minor"

    rev=$(db_get $key)
    if [ "$rev" = "" ]; then
        rev=0
    fi
    echo $rev
}

db_get() {
    key=$1
    git fetch -f origin +refs/db/$key:refs/db/$key 2>/dev/null || true
    git cat-file -p refs/db/$key 2>/dev/null
}

main
