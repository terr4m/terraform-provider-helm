set windows-shell := ["pwsh", "-NoLogo", "-Command"]

default:
    just --list

setup: tidy
    {{ if os() == "windows" { 'New-Item -Name "dist" -ItemType "directory" -Force' } else { "mkdir -p ./dist" } }}

tidy:
    go mod tidy

update:
    go get -u ./...

fmt:
    golangci-lint fmt

lint:
    golangci-lint run --fix --timeout 120s

test:
    go test -v -cover -timeout=120s -parallel=10 ./...

testacc $KUBE_CONFIG_PATH="~/.kube/config":
    go test -v -cover -timeout 120m -run TestAcc ./...

build:
    go build -o ./dist -v ./...

[working-directory("tools")]
docs:
    go generate ./...

docs-fmt:
    rumdl fmt --fix .

docs-lint:
    rumdl check .
