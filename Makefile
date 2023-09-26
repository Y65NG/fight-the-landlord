BINARY_NAME = landlord

build:
	GOARCH=amd64 GOOS=darwin go build -o ./bin/${BINARY_NAME}-darwin-x64
	GOARCH=arm64 GOOS=darwin go build -o ./bin/${BINARY_NAME}-darwin-arm
	GOARCH=amd64 GOOS=linux go build -o ./bin/${BINARY_NAME}-linux
	GOARCH=amd64 GOOS=windows go build -o ./bin/${BINARY_NAME}-windows

run: build
	./bin/${BINARY_NAME}-darwin-arm

clean:
	go clean
	rm ./bin/${BINARY_NAME}-darwin-x64
	rm ./bin/${BINARY_NAME}-darwin-arm
	rm ./bin/${BINARY_NAME}-linux
	rm ./bin/${BINARY_NAME}-windows

test:
	go test ./...

test_coverage:
	go test ./... -coverprofile=coverage.out

dep:
	go mod download

vet:
	go vet


