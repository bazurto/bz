#!/bin/bash

main() {
    major_minor=$(cat VERSION)
    revision=$(bump_revision $major_minor)
    echo "$major_minor.$revision"
}

#
# Usage:
#
# bump_revision 0.0
#   0
# bump_revision 0.0
#   1
# bump_revision 0.0
#   2
#
bump_revision() {
    major_mino=$1
    key="rev-$major_minor"

    rev=$(db_get $key)
    if [ "$rev" = "" ]; then
        rev=0
        db_set $key $rev
    else
        let rev=rev+1
        db_set $key $rev
    fi
    echo $rev
}

db_set() {
    key=$1
    value=$2
    git update-ref refs/db/$key $(echo $value | git hash-object -w --stdin)
    git push origin +refs/db/$key:refs/db/$key 2>/dev/null
}

db_get() {
    key=$1
    git fetch -f origin +refs/db/$key:refs/db/$key 2>/dev/null || true
    git cat-file -p refs/db/$key 2>/dev/null
}

main
