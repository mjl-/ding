#!/bin/bash
set -euo pipefail

export NAME=${NAME:-$(basename $(pwd))}

export VERSION=${VERSION:-$(git describe --tags | sed 's/^v//')}
export BRANCH=${BRANCH:-$(git show-ref | grep $(git show-ref -s -- HEAD) | sed 's|.*/\(.*\)|\1|' | grep -v HEAD | sort | uniq)}
export COMMIT=${COMMIT:-$(git rev-parse HEAD)}
export TAG=${TAG:-$(git describe --exact-match --tags 2>/dev/null)}
export GOVERSION=${GOVERSION:-$(go version | cut -f3 -d' ')}
export BUILDID=${BUILDID:-$(printf %x $(date +%s))}
echo version: $VERSION

make clean
make build
make frontend
(cd assets && zip -qr0 ../assets.zip .)

function release() {
	GOOS=$1
	GOARCH=$2
	SUFFIX=
	if test $GOOS = 'windows'; then
		SUFFIX=.exe
	fi
	DEST=local/$NAME-$VERSION-$GOOS-$GOARCH-$GOVERSION-$BUILDID$SUFFIX
	GOOS=$GOOS GOARCH=$GOARCH go build -ldflags "-X main.version=$VERSION -X main.vcsCommitHash=$COMMIT -X main.vcsBranch=$BRANCH -X main.vcsTag=$TAG" -o $DEST
	cat assets.zip >>$DEST
	echo release: $NAME $GOOS $GOARCH $GOVERSION $DEST
}

release linux amd64
release linux 386
release linux arm64
release darwin amd64
release openbsd amd64
