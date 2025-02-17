#!/bin/bash

find . -type f -name "*.pb.go" -exec rm {} \;

for d in proto/*/; do
    for f in "$d"*.proto; do
        protoc --proto_path="$d" --go_out="../" "$f"
    done
done
