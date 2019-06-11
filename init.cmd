@echo off

IF NOT DEFINED MYSRCROOT SET MYSRCROOT=%CD%

SET WORKSPACE=%MYSRCROOT%\workspace
SET GOPATH=%WORKSPACE%;%GOPATH%
SET PATH=%WORKSPACE%\bin;%PATH%

IF NOT EXIST %WORKSPACE%\src\microsoft.com                      MKDIR %WORKSPACE%\src\microsoft.com
IF NOT EXIST %WORKSPACE%\src\microsoft.com\github-events-spider MKLINK /D %WORKSPACE%\src\microsoft.com\github-events-spider %MYSRCROOT%

SET MYTOP=%WORKSPACE%\src\microsoft.com\github-events-spider
SET MYROOT=%WORKSPACE%

ECHO Updating go modules...
CD %MYTOP%
SET GO111MODULE=on
go mod download
