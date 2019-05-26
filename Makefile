run: build
	./ding serve -dbmigrate=false local/local.conf

run-root: build
	sudo sh -c 'umask 027; ./ding serve -dbmigrate=false -listen localhost:6086 -listenwebhook localhost:6087 local/local-root.conf'

build:
	go build
	go vet
	PATH=$(PATH):$(PWD)/node_modules/.bin go run fabricate/*.go -- install
	go run vendor/github.com/mjl-/sherpadoc/cmd/sherpadoc/*.go Ding >assets/ding.json

frontend:
	PATH=$(PATH):$(PWD)/node_modules/.bin go run fabricate/*.go -- install

test:
	golint
	go test -race -coverprofile cover.out -- local/local-test.conf
	go tool cover -html=cover.out -o cover.html

clean:
	go clean
	-rm -r assets assets.zip 2>/dev/null
	go run fabricate/*.go -- clean

setup:
	-mkdir -p node_modules/.bin
	npm install jshint@2.9.3 node-sass@4.7.2
