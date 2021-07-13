#!/bin/bash

set -e
#set -x

VOID_PACKAGES=$(realpath ../../void-packages)

function cleanup {
    rm -f $tmp $template_modified
}

mkdir -p /tmp/x10
tmp=/tmp/x10/voidpkg.out.$(uuidgen)
template_modified=/tmp/x10/voidpkg.template.$(uuidgen)
trap cleanup EXIT

sed -e 's/${version}/\\${X10_META_VERSION}/g' $VOID_PACKAGES/srcpkgs/$1/template >$template_modified

source $VOID_PACKAGES/common/environment/fetch/misc.sh
source $VOID_PACKAGES/common/environment/setup/options.sh
source $VOID_PACKAGES/common/xbps-src/shutils/common.sh

set -a
source $template_modified
set_build_options
source $template_modified
set +a

declare -A layermap=(
    [gnu-configure]=gnu_configure
)

declare -A stagemap=(
    [check]=test
)

if [[ -z $build_style ]]; then
    build_style=base
fi

if [[ -n ${layermap[$build_style]} ]]; then
    build_style=${layermap[$build_style]}
fi

export maintainer="$(git config user.name) <$(git config user.email)>"

echo "{}" >$tmp
yq -Yi '{
    layers: [env.build_style],
    package: {
        meta: {
            name: ("xbps/" + env.pkgname),
            version: env.version,
            revision: 1,
            maintainer: env.maintainer,
            homepage: env.homepage,
            license: env.license,
            description: env.short_desc
        }
    }
}' $tmp

i=0
echo "$distfiles" | while read distfile; do
    export distfile
    yq -Yi ".package.sources[$i].url = env.distfile" $tmp
    i=$((i+1))
done

i=0
echo "$checksum" | while read sum; do
    export sum
    yq -Yi ".package.sources[$i].checksum = env.sum" $tmp
    i=$((i+1))
done

if [[ -n $configure_args ]]; then
    yq -Yi '.package.environment.CONFIGURE_ARGS = (env.configure_args | gsub("\n"; ""))' $tmp
fi

### dependencies

for hostmakedepend in $hostmakedepends; do
    export hostmakedepend
    yq -Yi '.package.depends.hostbuild += [("xbps/" + env.hostmakedepend)]' $tmp
done

for makedepend in $makedepends; do
    export makedepend
    yq -Yi '.package.depends.build += [("xbps/" + env.makedepend)]' $tmp
done

for checkdepend in $checkdepends; do
    export checkdepend
    yq -Yi '.package.depends.test += [("xbps/" + env.checkdepend)]' $tmp
done

for depend in $depends; do
    export depend
    yq -Yi '.package.depends.run += [("xbps/" + env.depend)]' $tmp
done

### custom scripts

for f in $(typeset -F); do
    case "$f" in
        pre_*)
            export stage="${f#pre_}"
            if [[ -n ${stagemap[$stage]} ]]; then
                stage=${stagemap[$stage]}
            fi
            export body=$(declare -f "$f" | sed '1,2d;$d')
            body=$(echo "$body" | sed -e 's/^    //g')
            if [ $(echo "$body" | wc -l) -gt 2 ]; then
                yq -Yi '.package.stages[env.stage].prescript = [env.body + "\n", "__yq_style_0_|__"]' $tmp
            else
                yq -Yi '.package.stages[env.stage].prescript = [env.body]' $tmp
            fi
            ;;
        post_*)
            export stage="${f#post_}"
            if [[ -n ${stagemap[$stage]} ]]; then
                stage=${stagemap[$stage]}
            fi
            export body=$(declare -f "$f" | sed '1,2d;$d')
            body=$(echo "$body" | sed -e 's/^    //g')
            if [ $(echo "$body" | wc -l) -gt 2 ]; then
                yq -Yi '.package.stages[env.stage].postscript = [env.body + "\n", "__yq_style_0_|__"]' $tmp
            else
                yq -Yi '.package.stages[env.stage].postscript = [env.body]' $tmp
            fi
            ;;
        do_*)
            export stage="${f#do_}"
            if [[ -n ${stagemap[$stage]} ]]; then
                stage=${stagemap[$stage]}
            fi
            export body=$(declare -f "$f" | sed '1,2d;$d')
            body=$(echo "$body" | sed -e 's/^    //g')
            yq -Yi '.package.stages[env.stage].script = env.body + "\n"' $tmp
            # echo -n 'script' | shasum -a224 | cut -d' ' -f1 | xxd -r -p | base64
            yq -Yi '.package.stages[env.stage]["__yq_style_rYT6kwx+fJKY5CSNbWrNIFpj9Y7o/GDWOEzOVw==__"] = "|"' $tmp
    esac
done


echo "# auto-generated from void-packages/$1"
cat $tmp
