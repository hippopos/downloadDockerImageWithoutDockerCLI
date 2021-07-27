CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o nodocker-linux-amd64 .
CGO_ENABLED=0 GOOS=darwin go build -a -installsuffix cgo -o nodocker-darwin .
CGO_ENABLED=0 GOOS=windows go build -a -installsuffix cgo -o nodocker.exe .