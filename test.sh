#!/bin/bash

shopt -s extglob
home=$(pwd)

# If we're not under a "src" directory we're (probably) on the CI server.
# export GOPATH and cd to the right location
if [[ $home != *"src"* ]]; then
  export GOPATH=${home}

  dir=$(git config --get remote.origin.url)
  dir=${dir#http://}   # remove leading http://
  dir=${dir#https://}  # remove leading https://
  dir=${dir%.git}      # remove trailing .git
  dir="src/${dir}"     # add src/ prefix

  cd ${dir}
fi

DIRS=$(go list ./... | grep -v '\/vendor\/')

printf "\nGo dirs:\n${DIRS}\n\n"

if [[ -z $DIRS ]]; then
  echo "No Go dirs found."
  exit 255
fi

for dir in $DIRS; do
  cd $GOPATH/src/${dir}

  echo "Running tests for ${dir}..."
  if [ -f cover.out ]; then
    rm cover.out
  fi

  echo "go test -v -timeout 3m --race -cpu 1"
  go test -v -timeout 3m --race -cpu 1
  if [ $? -ne 0 ]; then
    exit 255
  fi

  echo "go test -v -timeout 3m --race -cpu 4"
  go test -v -timeout 3m --race -cpu 4
  if [ $? -ne 0 ]; then
    exit 255
  fi

  echo "go test -v -timeout 3m -coverprofile cover.out"
  go test -v -timeout 3m -coverprofile cover.out
  if [ $? -ne 0 ]; then
    exit 255
  fi
  
  printf "\n"
done

echo "Success."
