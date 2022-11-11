#!/bin/bash

set -e

# Get latest release assets from GitHub API
assets=$(curl -H "Accept: application/vnd.github.v3+json" \
    https://api.github.com/repos/bytecodealliance/wasmtime/releases/latest)

osarch=$(uname -ms)
unsupported="Unsupported architecture "

function get_asset {
    asset=$(echo $assets | gojq -r '.assets.[] | .browser_download_url' | grep "$1-c-api.tar.xz")
    echo $asset
    curl -L -o archive.tar.xz $asset
    tar --strip-components=2 \
        --wildcards \
        -xf archive.tar.xz \
        "*libwasmtime.a"
}

if [ "$osarch" == "Linux x86_64" ]
then
    get_asset "x86_64-linux"
elif [ "$osarch" == "Linux aarch64" ] # for some reason this is aarch64...
then
    get_asset "aarch64-linux"
elif [ "$osarch" == "Darwin x86_64" ]
then
    get_asset "x86_64-macos"
elif [ "$osarch" == "Darwin arm64" ] # ...and this is arm64
then
    echo $unsupported $osarch
    exit 1
else
    echo $unsupported $osarch
    exit 1
fi
