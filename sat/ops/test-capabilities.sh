#!/bin/bash

docker run -it \
    --network=suborbital_scn \
    -v "${PWD}/examples/http-cap:/tmp" \
    -e SAT_HTTP_PORT=8079 \
    -e SAT_CONTROL_PLANE=scc-control-plane:8081 \
    -p 8079:8079 \
    suborbital/sat:dev sat "/tmp/http-cap.wasm"
