@echo off
setlocal
cd /d "%~dp0"

echo Cleaning up old processes...
taskkill /F /IM go.exe >nul 2>&1
taskkill /F /IM main.exe >nul 2>&1
timeout /t 2 >nul

echo [1/6] Starting Auth Service...
set AUTH_SERVICE_PORT=8081
set DATA_DIR=%~dp0auth-service\data
start "Auth Service" cmd /c "cd auth-service && go run ./cmd/main.go"

echo [2/6] Starting Product Service...
set PRODUCT_SERVICE_PORT=8082
set DATA_DIR=%~dp0product-service\data
start "Product Service" cmd /c "cd product-service && go run ./cmd/main.go"

echo [3/6] Starting Address Service...
set ADDRESS_SERVICE_PORT=8083
set DATA_DIR=%~dp0address-service\data
start "Address Service" cmd /c "cd address-service && go run ./cmd/main.go"

echo [4/6] Starting Order Service...
set ORDER_SERVICE_PORT=8084
set PRODUCT_SERVICE_URL=http://localhost:8082
set DATA_DIR=%~dp0order-service\data
start "Order Service" cmd /c "cd order-service && go run ./cmd/main.go"

echo [5/6] Starting CDN Service...
set CDN_SERVICE_PORT=8085
set STATIC_DIR=%~dp0cdn-service\static
start "CDN Service" cmd /c "cd cdn-service && go run ./cmd/main.go"

echo Waiting 6 seconds before starting API Gateway...
timeout /t 6 >nul

echo [6/6] Starting API Gateway...
set GATEWAY_PORT=8080
set AUTH_SERVICE_URL=http://localhost:8081
set PRODUCT_SERVICE_URL=http://localhost:8082
set ADDRESS_SERVICE_URL=http://localhost:8083
set ORDER_SERVICE_URL=http://localhost:8084
start "API Gateway" cmd /c "cd api-gateway && go run ./cmd/main.go"

echo All microservices started in separate windows!
echo DO NOT CLOSE THE BLACK WINDOWS if you want the backend to keep running.
pause
