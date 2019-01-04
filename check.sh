#!/bin/bash

# go get -u -v golang.org/x/tools/cmd/goimports
# go get -u github.com/kisielk/errcheck
# go get -u github.com/golang/lint/golint
# go get -u github.com/mdempsky/unconvert
# go get -u github.com/client9/misspell/cmd/misspell
# go get -u github.com/gordonklaus/ineffassign
# go get -u honnef.co/go/tools/cmd/staticcheck
# go get -u github.com/fzipp/gocyclo

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

# convert path to lowercase
# prevent windows/system32/find.exe from being the 'find' we use
uname=$(uname)
if [[ $uname == "MSYS_NT"* ]] || [[ $uname == "MINGW"* ]]; then
  echo "Running on MinGW - ${uname}."
  PATH=$(echo $PATH | tr '[:upper:]' '[:lower:]') # convert path to all lowercase
  PATH=${PATH/\/c\/windows\/system32:/}           # remove /c/windows/system32:
else
  echo "Not running on MinGW - ${uname}."
fi

FILES=$(find . -type f -iname "*.go"|grep -v '\/vendor\/')
DIRS=$(go list ./... | grep -v '\/vendor\/')

printf "\nGo files:\n${FILES}\n\n"
printf "Go dirs:\n${DIRS}\n\n"

if [[ -z $FILES ]]; then
  echo "No Go files found."
  exit 255
fi

if [[ -z $DIRS ]]; then
  echo "No Go dirs found."
  exit 255
fi

echo "Running static analysis..."

hasErr=0

echo "- Checking gofmt..."
fmtRes=$(gofmt -l -s -d $FILES)
if [ -n "${fmtRes}" ]; then
  echo "gofmt checking failed: ${fmtRes}"
  hasErr=1
fi

echo "- Checking goimports..."
impRes=$(goimports -l -d $FILES)
if [ -n "${impRes}" ]; then
  echo "goimports checking failed: ${impRes}"
  hasErr=1
fi

echo "- Checking errcheck..."
for dir in $DIRS; do
  errRes=$(errcheck -blank -asserts ${dir})
  if [ $? -ne 0 ]; then
    echo "errcheck checking failed: ${errRes}"
    hasErr=1
  elif [ -n "${errRes}" ]; then
    echo "errcheck checking failed: ${errRes}"
    hasErr=1
  fi
done

echo "- Checking go tool vet -all -shadow..."
for path in $FILES; do
  go tool vet -all -shadow ${path}
  if [ $? -ne 0 ]; then
    hasErr=1
  fi
done

echo "- Checking golint..."
lintError=0
for path in $FILES; do
  lintRes=$(golint -set_exit_status ${path})
  if [ -n "${lintRes}" ]; then
    echo "golint checking ${path} failed: ${lintRes}"
    hasErr=1
  fi
done

echo "- Checking unconvert..."
for dir in $DIRS; do
  unconvertRes=$(unconvert ${dir})
  if [ $? -ne 0 ]; then
    echo "unconvert checking failed: ${unconvertRes}"
    hasErr=1
  elif [ -n "${unconvertRes}" ]; then
    echo "unconvert checking failed: ${unconvertRes}"
    hasErr=1
  fi
done

echo "- Checking misspell..."
misspellRes=$(misspell -error $FILES)
if [ $? -ne 0 ]; then
  echo "misspell checking failed: ${misspellRes}"
  hasErr=1
elif [ -n "${misspellRes}" ]; then
  echo "misspell checking failed: ${misspellRes}"
  hasErr=1
fi

echo "- Checking ineffassign..."
for file in $FILES; do
  ineffassignRes=$(ineffassign ${file})
  if [ $? -ne 0 ]; then
    echo "ineffassign checking failed: ${ineffassignRes}"
    hasErr=1
  elif [ -n "${ineffassignRes}" ]; then
    echo "ineffassign checking failed: ${ineffassignRes}"
    hasErr=1
  fi
done

echo "- Checking staticcheck..."
for dir in $DIRS; do
  staticcheckRes=$(staticcheck ${dir})
  if [ $? -ne 0 ]; then
    echo "staticcheck checking failed: ${staticcheckRes}"
    hasErr=1
  elif [ -n "${staticcheckRes}" ]; then
    echo "staticcheck checking failed: ${staticcheckRes}"
    hasErr=1
  fi
done

echo "- Checking gocyclo..."
gocycloRes=$(gocyclo -over 15 $FILES)
if [ -n "${gocycloRes}" ]; then
  echo "gocyclo warning: ${gocycloRes}"
fi

if [ $hasErr -ne 0 ]; then
  printf "\nLint errors."
  exit 255
fi

printf "\nSuccess."
