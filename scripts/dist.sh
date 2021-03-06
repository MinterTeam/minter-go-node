#!/usr/bin/env bash
set -e

# WARN: non hermetic build (people must run this script inside docker to
# produce deterministic binaries).

# Get the version from the environment, or try to figure it out.
if [ -z $VERSION ]; then
	VERSION=$(awk -F\" '/Version =/ { print $2; exit }' < version/version.go)
fi
if [ -z "$VERSION" ]; then
    echo "Please specify a version."
    exit 1
fi
echo "==> Building version $VERSION..."

# Delete the old dir
echo "==> Removing old directory..."
rm -rf build/pkg
mkdir -p build/pkg

# Get the git commit
GIT_COMMIT="$(git rev-parse --short=8 HEAD)"
GIT_IMPORT="github.com/MinterTeam/minter-go-node/version"

#packr

# Build!
# ldflags: -s Omit the symbol table and debug information.
#	         -w Omit the DWARF symbol table.
echo "==> Building for mac os..."
CGO_ENABLED=1 go build -tags "minter gcc" -ldflags "-s -w -X ${GIT_IMPORT}.GitCommit=${GIT_COMMIT}" -o "build/pkg/darwin_amd64/minter" ./cmd/minter

echo "==> Building for linux in docker"
docker run -t -v ${PWD}:/go/src/github.com/MinterTeam/minter-go-node/ -i minter-builder-1:latest sh -c "CGO_ENABLED=1 go build -tags 'minter gcc' -ldflags '-s -w -X ${GIT_IMPORT}.GitCommit=${GIT_COMMIT}' -o 'build/pkg/linux_amd64/minter' ./cmd/minter/"

# Zip all the files.
echo "==> Packaging..."
for PLATFORM in $(find ./build/pkg -mindepth 1 -maxdepth 1 -type d); do
	OSARCH=$(basename "${PLATFORM}")
	echo "--> ${OSARCH}"

	pushd "$PLATFORM" >/dev/null 2>&1
	zip "../${OSARCH}.zip" ./*
	popd >/dev/null 2>&1
done

# Add "minter" and $VERSION, $GIT_COMMIT prefix to package name.
rm -rf ./build/dist
mkdir -p ./build/dist
for FILENAME in $(find ./build/pkg -mindepth 1 -maxdepth 1 -type f); do
  FILENAME=$(basename "$FILENAME")
	cp "./build/pkg/${FILENAME}" "./build/dist/minter_${VERSION}_${GIT_COMMIT}_${FILENAME}"
done

# Make the checksums.
pushd ./build/dist
shasum -a256 ./* > "./minter_${VERSION}_${GIT_COMMIT}_SHA256SUMS"
popd

# Done
echo
echo "==> Results:"
ls -hl ./build/dist

exit 0
