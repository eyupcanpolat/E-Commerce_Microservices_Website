@echo off
setlocal
cd /d "%~dp0"

echo Starting Frontend Server on port 3000...
go run serve.go
