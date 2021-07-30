# downloadDockerImageWithoutDockerCLI

download docker image from dockerHub or private registry without dockerCLI,
Support Linux/Macos/Windows

Usage:

```shell script
./nodocker pull busybox

./nodocker pull localhost:5000/busybox  --registry-user username --registry-password password
```

Build from src
```shell script

go mod download

CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o nodocker .
```
