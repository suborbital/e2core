# /bin/bash

set -e

LIB="/tmp/wasmerio/linux-amd64/libwasmer.so"

if [ $(uname -m) != x86_64 ]
then
	LIB="/tmp/wasmerio/linux-aarch64/libwasmer.so"
fi

echo "using $LIB"

cp $LIB /usr/local/lib

rm -rf /tmp/wasmerio