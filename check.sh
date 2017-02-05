#!/bin/bash

# go get -u github.com/kisielk/errcheck
# go get -u github.com/golang/lint/golint
# go get -u honnef.co/go/tools/cmd/gosimple
# go get -u honnef.co/go/tools/cmd/unused
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

echo "- Checking errcheck..."

echo "(*os.File).Close">errcheck_excludes.txt
echo "(io.Closer).Close">>errcheck_excludes.txt

for dir in $DIRS; do
  errRes=$(errcheck -blank -asserts -exclude errcheck_excludes.txt ${dir})
  if [ $? -ne 0 ]; then
    echo "errcheck checking failed: ${errRes}"
    hasErr=1
  elif [ -n "${errRes}" ]; then
    echo "errcheck checking failed: ${errRes}"
    hasErr=1
  fi
done

rm errcheck_excludes.txt

echo "- Checking govet..."

for dir in $DIRS; do
  go vet ${dir}
  if [ $? -ne 0 ]; then
    hasErr=1
  fi
done

echo "- Checking govet -shadow..."
for path in $FILES; do
  go tool vet -shadow ${path}
  if [ $? -ne 0 ]; then
    hasErr=1
  fi
done

echo "- Checking golint..."
lintError=0
for path in $FILES; do
  lintRes=$(golint ${path})
  if [ -n "${lintRes}" ]; then
    echo "golint checking ${path} failed: ${lintRes}"
    hasErr=1
  fi
done

echo "- Checking gosimple..."
for dir in $DIRS; do
  gosimpleRes=$(gosimple ${dir})
  if [ $? -ne 0 ]; then
    echo "gosimple checking failed: ${gosimpleRes}"
    hasErr=1
  elif [ -n "${gosimpleRes}" ]; then
    echo "gosimple checking failed: ${gosimpleRes}"
    hasErr=1
  fi
done

echo "- Checking unused..."
for dir in $DIRS; do
  unusedRes=$(unused ${dir})
  if [ $? -ne 0 ]; then
    echo "unused checking failed: ${unusedRes}"
    hasErr=1
  elif [ -n "${unusedRes}" ]; then
    echo "unused checking failed: ${unusedRes}"
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
misspellRes=$(misspell $FILES)
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
