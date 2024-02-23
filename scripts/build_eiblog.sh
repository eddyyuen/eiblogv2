#!/usr/bin/env sh
# prepare dir ./bin
app=eiblog
_arch="$(go env GOARCH)"
_os="linux"
# tar platform
_target="$app-$_os-$_arch.tar.gz"
CGO_ENABLED=1 GOOS=$_os GOARCH=$_arch go build -x -v -tags prod -ldflags '-s -w -linkmode "external" -extldflags "-static-libgcc -s" -linkshared' -o eiblog "./cmd/$app"
tar czf $_target conf website assets eiblog
