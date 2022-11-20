now = $(shell date +'%Y-%m-%dT%TZ')
commit = $(shell if [ ! -d .git ]; then echo "unknown"; else git rev-parse --short HEAD; fi)
var_path = github.com/suborbital/subo/subo/release
RELEASE_FLAGS = "-X $(var_path).CommitHash=$(commit)\
 -X $(var_path).BuildTime=$(now)"