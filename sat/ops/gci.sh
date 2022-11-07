#!/bin/bash

gci write -s Standard -s Default -s "Prefix(github.com/suborbital)" -s "Prefix(github.com/suborbital/e2core/sat)" --NoInlineComments --NoPrefixComments main.go $(find {sat,constd}/ -type f -name '*.go')
