run: build
	./ding -loglevel debug serve local/local.conf

run-root: build
	sudo sh -c 'umask 027; ./ding -loglevel debug serve -listen localhost:6186 -listenwebhook localhost:6187 -listenadmin localhost:6188 local/local-root.conf'

build: node_modules/.bin/tsc
	CGO_ENABLED=0 go build
	CGO_ENABLED=0 go vet
	CGO_ENABLED=0 go run vendor/github.com/mjl-/sherpadoc/cmd/sherpadoc/*.go -adjust-function-names none Ding >web/ding.json
	./gents.sh web/ding.json api.ts
	./genlicense.sh
	./tsc.sh web/ding.js dom.ts api.ts ding.ts
	CGO_ENABLED=0 go build # build with generated files

check:
	CGO_ENABLED=0 go vet
	GOARCH=386 CGO_ENABLED=0 go vet
	CGO_ENABLED=0 staticcheck
	golint

tswatch:
	bash -c 'while true; do inotifywait -q -e close_write *.ts; make web/ding.js; done'

web/ding.js: node_modules/.bin/tsc dom.ts api.ts ding.ts
	./tsc.sh web/ding.js dom.ts api.ts ding.ts

node_modules/.bin/tsc:
	-mkdir -p node_modules/.bin
	npm ci --ignore-scripts

install-js:
	-mkdir -p node_modules/.bin
	npm install --ignore-scripts --save-dev --save-exact typescript@5.1.6

# note: running as root (with umask 0022) tests the privsep paths
test:
	CGO_ENABLED=0 go test -shuffle=on -coverprofile cover.out
	go tool cover -html=cover.out -o cover.html

test-race:
	CGO_ENABLED=1 go test -shuffle=on -race -coverprofile cover.out
	go tool cover -html=cover.out -o cover.html

test-gotoolchains:
	DING_TEST_GOTOOLCHAINS=yes CGO_ENABLED=0 go test -coverprofile cover.out
	go tool cover -html=cover.out -o cover.html

fmt:
	gofmt -w -s *.go

clean:
	CGO_ENABLED=0 go clean
