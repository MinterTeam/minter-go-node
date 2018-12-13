#!/usr/bin/env bash
set -e

## check if GOPATH is set
if [ -z ${GOPATH+x} ]; then
	echo "please set GOPATH (https://github.com/golang/go/wiki/SettingGOPATH)"
	exit 1
fi

mkdir -p "$GOPATH/src/github.com"
cd "$GOPATH/src/github.com" || exit 1

installFromGithub() {
	repo=$1
	commit=$2
	# optional
	subdir=$3
	echo "--> Installing $repo ($commit)..."
	if [ ! -d "$repo" ]; then
		mkdir -p "$repo"
		git clone "https://github.com/$repo.git" "$repo"
	fi
	if [ ! -z ${subdir+x} ] && [ ! -d "$repo/$subdir" ]; then
		echo "ERROR: no such directory $repo/$subdir"
		exit 1
	fi
	pushd "$repo" && \
		git fetch origin && \
		git checkout -q "$commit" && \
		if [ ! -z ${subdir+x} ]; then cd "$subdir" || exit 1; fi && \
		go install && \
		if [ ! -z ${subdir+x} ]; then cd - || exit 1; fi && \
		popd || exit 1
	echo "--> Done"
	echo ""
}

installFromGithub mitchellh/gox 51ed453898ca5579fea9ad1f08dff6b121d9f2e8
installFromGithub golang/dep 22125cfaa6ddc71e145b1535d4b7ee9744fefff2 cmd/dep
installFromGithub alecthomas/gometalinter 17a7ffa42374937bfecabfb8d2efbd4db0c26741
installFromGithub gogo/protobuf 61dbc136cf5d2f08d68a011382652244990a53a9 protoc-gen-gogo
installFromGithub square/certstrap e27060a3643e814151e65b9807b6b06d169580a7
installFromGithub rogpeppe/go-internal 3da46a1d085e759e4ad3c247ed87fe056d62037a modfile
installFromGithub gobuffalo/envy 801d7253ade1f895f74596b9a96147ed2d3b087e
installFromGithub gobuffalo/syncx 558ac7de985fc4f4057bff27c7bdf99e92fe0750
installFromGithub gobuffalo/packd c49825f8f6f458178e241c985b37f979045e6879
installFromGithub markbates/oncer bf2de49a0be218916e69a11d22866e6cd0a560f2
installFromGithub gobuffalo/packr 0bf68f085c6ef75e9f2403052b9e79ddd42a660e
