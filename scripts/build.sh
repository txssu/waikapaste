#!/usr/bin/env bash

SCRIPTS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
DIR="$SCRIPTS_DIR/.."
BUILD_DIR="$DIR/build"

echo "Clean"
rm -rf $BUILD_DIR/*

echo "Build"
cd $DIR && env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $BUILD_DIR/wpaste
