#! /bin/bash

# subo build .
ATMO_HEADLESS=true ATMO_HTTP_PORT=8080 ATMO_LOG_LEVEL=error atmo ./runnables.wasm.zip &
ATMO_PID=$!

sleep 3

hey -c 50 -t 20 -n 500000 -m POST -d 'my friend' http://localhost:8080/com.suborbital.perf/default/rs-perf/v0.1.0
hey -c 50 -t 20 -n 500000 -m POST -d 'my friend' http://localhost:8080/com.suborbital.perf/default/js-perf/v0.1.0
# hey -c 40 -t 20 -o csv localhost:8080/com.suborbital.perf::default#go-perf@v0.0.1
# hey -c 40 -t 20 -o csv localhost:8080/com.suborbital.perf::default#grain-perf@v0.0.1

killall -9 atmo

# launch node server
# test node

# kill node

# build rust server
# launch rust server

# test rust server

# summarize results