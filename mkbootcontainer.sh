#!/bin/bash

set -e
set -x

cid=$(buildah from scratch)
uuid=$(uuidgen)
mkdir -p /tmp/x10/mkcontainer.$uuid
function cleanup {
    buildah rm $cid
    rm -rf /tmp/x10/mkcontainer.$uuid
}
trap cleanup EXIT

export X10_TARGETDIR=../targetdir_boot

for f in $(cat bootstrap); do
    go run . install "$f" /tmp/x10/mkcontainer.$uuid
done
buildah add $cid /tmp/x10/mkcontainer.$uuid /
buildah commit $cid digitalis_bootstrap
