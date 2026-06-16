@echo off
REM build.bat: cross-compile kload for all supported platforms
REM Produces stripped binaries (-s -w) in the .\dist directory.
REM Usage: build.bat [version]   e.g.  build.bat 1.0.0

setlocal enabledelayedexpansion

set APP=kload
set ENTRY=.\main.go
set DIST=dist

REM Version from first arg, default "dev"
set VERSION=%1
if "%VERSION%"=="" set VERSION=dev

set LDFLAGS=-s -w -X main.version=%VERSION%

echo Building %APP% %VERSION%

REM Clean dist
if exist "%DIST%" rmdir /s /q "%DIST%"
mkdir "%DIST%"

REM ++ Linux amd64 ++
echo   -^> linux/amd64
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -ldflags="%LDFLAGS%" -o "%DIST%\%APP%-linux-amd64" "%ENTRY%"

REM ++ Linux arm64 ++
echo   -^> linux/arm64
set GOOS=linux
set GOARCH=arm64
go build -ldflags="%LDFLAGS%" -o "%DIST%\%APP%-linux-arm64" "%ENTRY%"

REM ++ macOS amd64 ++
echo   -^> darwin/amd64
set GOOS=darwin
set GOARCH=amd64
go build -ldflags="%LDFLAGS%" -o "%DIST%\%APP%-darwin-amd64" "%ENTRY%"

REM ++ macOS arm64 ++
echo   -^> darwin/arm64
set GOOS=darwin
set GOARCH=arm64
go build -ldflags="%LDFLAGS%" -o "%DIST%\%APP%-darwin-arm64" "%ENTRY%"

REM ++ Windows amd64 ++
echo   -^> windows/amd64
set GOOS=windows
set GOARCH=amd64
go build -ldflags="%LDFLAGS%" -o "%DIST%\%APP%-windows-amd64.exe" "%ENTRY%"

echo.
echo Done. Binaries in .\%DIST%:
dir "%DIST%"

endlocal