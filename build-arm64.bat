@echo off
set GOOS=linux
set GOARCH=arm64
set CGO_ENABLED=0
go build -ldflags "-s -w" -o rail-notifier-arm64 ./cmd/main/
if %errorlevel% equ 0 (
    echo Build successful: rail-notifier-arm64
) else (
    echo Build failed!
)
exit
