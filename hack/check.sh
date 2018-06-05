#!/bin/sh

set -e
Packages="$(go list ./... | grep -v -E 'vendor' | xargs echo)"

echo "Checking format..."
go list ./... | grep -v vendor | sed -e s=github.com/oif/gokit/=./= | xargs -n 1 gofmt -l

echo "Checking govet..."
go vet $Packages

set -e
echo "Testing..."

function runtest {
  bash -c "umask 0; PATH=$GOROOT/bin:$(pwd)/bin:$PATH go test $@ --cover"
}

for t in ${Packages}; do
  runtest "${t}"
done
