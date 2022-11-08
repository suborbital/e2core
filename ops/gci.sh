#!/bin/bash

gci write -s Standard -s Default -s "Prefix(github.com/suborbital)" $(find {bus,command,common,e2core,fqfn,options,sat,scheduler,server,signaler,syncer}/ -type f -name '*.go')
