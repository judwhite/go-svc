@echo off
rem run this script as admin

if not exist example.exe (
    echo Build the example before installing by running "go build"
    goto :exit
)

sc create go-svc-example binpath= "%CD%\example.exe" start= auto DisplayName= "go-svc-example"
sc description go-svc-example "go-svc-example"
sc start go-svc-example
sc query go-svc-example

echo Check example.log

:exit
