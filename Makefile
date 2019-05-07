run: build
	./ding serve local/config.json

run-root: build
	sudo ./ding serve local/config.json


build:
	go build
	go run fabricate/*.go -- install
	go run vendor/github.com/mjl-/sherpadoc/cmd/sherpadoc/*.go Ding >assets/ding.json

frontend:
	go run fabricate/*.go -- install

test:
	go vet ./...
	golint
	go test -cover . -- local/config-test.json

coverage:
	go test -coverprofile=coverage.out -test.outputdir . -- local/config-test.json
	go tool cover -html=coverage.out

fmt:
	go fmt ./...

release:
	-mkdir local 2>/dev/null
	(cd assets && zip -qr0 ../assets.zip .)
	env GOOS=linux GOARCH=amd64 ./release.sh
	env GOOS=linux GOARCH=386 ./release.sh
	env GOOS=linux GOARCH=arm GOARM=6 ./release.sh
	env GOOS=linux GOARCH=arm64 ./release.sh
	env GOOS=darwin GOARCH=amd64 ./release.sh
	env GOOS=openbsd GOARCH=amd64 ./release.sh

clean:
	go clean
	-rm -r assets assets.zip
	go run fabricate/*.go -- clean

setup:
	-mkdir -p node_modules/.bin
	npm install jshint@2.9.3 node-sass@4.7.2
