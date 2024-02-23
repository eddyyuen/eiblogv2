#!/usr/bin/env sh
# prepare dir ./bin
app=eiblog
_arch="$(go env GOARCH)"
_tag="custom"
_os="linux"
# tar platform
_target="$app-$_tag.$_os-$_arch.tar.gz"
GOOS=$_os GOARCH=$_arch go build -tags prod -ldflags '-extldflags "-static"' -o eiblog "./cmd/$app"
tar czf $_target conf website assets eiblog
