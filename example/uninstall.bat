@echo off
rem run this script as admin

net stop go-svc-example
sc delete go-svc-example
