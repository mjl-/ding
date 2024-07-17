export CGO_ENABLED=0
export GOFLAGS=-mod=vendor
export GOPROXY=off

PG=/usr/lib/postgresql/9.5

run: build
	./ding serve -dbmigrate=false local/local.conf

run-root: build
	sudo sh -c 'umask 027; ./ding serve -dbmigrate=false -listen localhost:6186 -listenwebhook localhost:6187 -listenadmin localhost:6188 local/local-root.conf'

build:
	go build
	go vet
	PATH=$(PATH):$(PWD)/node_modules/.bin go run fabricate/*.go -- install
	go run vendor/github.com/mjl-/sherpadoc/cmd/sherpadoc/*.go Ding >assets/ding.json

frontend:
	PATH=$(PATH):$(PWD)/node_modules/.bin go run fabricate/*.go -- install


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

test:
	golint
	CGO_ENABLED=1 go test -race -coverprofile cover.out -args local/local-test.conf
	go tool cover -html=cover.out -o cover.html

fmt:
	go fmt ./...

clean:
	go clean
	-rm -r assets assets.zip 2>/dev/null
	go run fabricate/*.go -- clean

setup:
	npm ci

setup0:
	-mkdir -p node_modules/.bin
	npm install --save-dev --save-exact jshint@2.9.3 sass@1.71.1
