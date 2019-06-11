@echo off

CD %MYTOP%

ECHO Building source tree...
SET GO111MODULE=on
go build -o .\bin\go-ycsb.exe .\cmd\go-ycsb