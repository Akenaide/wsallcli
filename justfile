default:
    @just --list

build:
    go build -o wsallcli .

test:
    go test ./...

lint:
    go vet ./...

run-rose: build
    ./wsallcli rose

docker-build:
    docker build -t wsallcli .

docker-run: docker-build
    docker run -v $(pwd)/data:/data wsallcli rose
