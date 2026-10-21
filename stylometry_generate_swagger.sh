#!/bin/sh
set -eu

cd "$(dirname "$0")"

created_mod=false
created_sum=false
if [ ! -e go.mod ]; then
  ln -s stylometry_go.mod go.mod
  created_mod=true
fi
if [ ! -e go.sum ]; then
  ln -s stylometry_go.sum go.sum
  created_sum=true
fi
trap 'if [ "$created_mod" = true ]; then rm -f go.mod; fi; if [ "$created_sum" = true ]; then rm -f go.sum; fi' 0

SWAG_BIN="${SWAG_BIN:-swag}"
"$SWAG_BIN" init \
  -g stylometry_main.go \
  -d ./stylometry_cmd/stylometry_stylometry,./stylometry_internal/stylometry_app/stylometry_handler,./stylometry_internal/stylometry_app/stylometry_serializer \
  --parseInternal \
  --parseGoList=false \
  -o stylometry_docs
