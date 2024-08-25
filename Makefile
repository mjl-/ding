export CGO_ENABLED=0
export GOFLAGS=-mod=vendor
export GOPROXY=off

PG=/usr/lib/postgresql/9.5

run: build
	./ding serve local/local.conf

run-root: build
	sudo sh -c 'umask 027; ./ding serve -listen localhost:6186 -listenwebhook localhost:6187 -listenadmin localhost:6188 local/local-root.conf'

build: frontend
	go build
	go vet
	PATH=$(PATH):$(PWD)/node_modules/.bin go run fabricate/*.go -- install
	go run vendor/github.com/mjl-/sherpadoc/cmd/sherpadoc/*.go Ding >assets/ding.json

frontend:
	PATH=$(PATH):$(PWD)/node_modules/.bin go run fabricate/*.go -- install

check:
	CGO_ENABLED=0 go vet
	GOARCH=386 CGO_ENABLED=0 go vet
	CGO_ENABLED=0 staticcheck
	golint

postgres-init:
	$(PG)/bin/initdb -D local/postgres95

postgres-makeuser:
	$(PG)/bin/createuser -h localhost -p 5437 --no-createdb --pwprompt ding
	$(PG)/bin/createdb -h localhost -p 5437 -O ding ding
	$(PG)/bin/createuser -h localhost -p 5437 --no-createdb --pwprompt ding_test
	$(PG)/bin/createdb -h localhost -p 5437 -O ding_test ding_test

postgres:
	$(PG)/bin/postgres -D local/postgres95 -p 5437 -k '' 2>&1 | tee local/postgres95/postgres.log

psql:
	$(PG)/bin/psql -h localhost -p 5437 -d ding

# note: running as root (with umask 0022) tests the privsep paths
test:
	CGO_ENABLED=0 go test -coverprofile cover.out
	go tool cover -html=cover.out -o cover.html

test-race:
	CGO_ENABLED=1 go test -race -coverprofile cover.out
	go tool cover -html=cover.out -o cover.html

test-gotoolchains:
	DING_TEST_GOTOOLCHAINS=yes CGO_ENABLED=0 go test -coverprofile cover.out
	go tool cover -html=cover.out -o cover.html

fmt:
	go fmt ./...

clean:
	go clean
	go run fabricate/*.go -- clean

setup:
	npm ci

setup0:
	-mkdir -p node_modules/.bin
	npm install --save-dev --save-exact jshint@2.9.3 sass@1.71.1
