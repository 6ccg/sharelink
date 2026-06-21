@echo off
setlocal

set "ROOT=%~dp0"
set "DATA_DIR=%ROOT%backend"

if /I "%~1"=="backend" goto backend
if /I "%~1"=="frontend" goto frontend
if /I "%~1"=="check" goto check

if not exist "%DATA_DIR%" mkdir "%DATA_DIR%"

echo ShareLink test runner
echo.
echo Backend:  http://localhost:8080
echo Frontend: http://localhost:5173/admin
echo Login:    url / test-password
echo DB:       backend\sharelink.db
echo.
echo This script is for local testing only.
echo Close the opened terminal windows to stop the services.
echo.

start "ShareLink Backend (test only)" cmd /k ""%~f0" backend"
start "ShareLink Frontend (test only)" cmd /k ""%~f0" frontend"

exit /b 0

:check
echo Checking ShareLink test runner dependencies...
where go >nul 2>nul || (echo Missing go in PATH.& exit /b 1)
where npm >nul 2>nul || (echo Missing npm in PATH.& exit /b 1)
if not exist "%ROOT%backend\data\ip2region.xdb" (
  echo Missing backend\data\ip2region.xdb.
  exit /b 1
)
if not exist "%ROOT%frontend\node_modules" (
  echo Missing frontend\node_modules. Run npm install in frontend first.
  exit /b 1
)
echo OK
exit /b 0

:backend
cd /d "%ROOT%backend"
set "NODE_ENV=development"
set "PORT=8080"
set "DB_TYPE=sqlite"
set "DB_DSN=%DATA_DIR%\sharelink.db"
set "DATA_DIR=%DATA_DIR%"
set "IP_DB_PATH=data\ip2region.xdb"
set "LOG_LEVEL=debug"
set "CORS_ALLOWED_ORIGINS=http://localhost:5173,http://127.0.0.1:5173"
set "INITIAL_ADMIN_PASSWORD=test-password"
set "JWT_SECRET=sharelink-test-jwt-secret-at-least-32-chars"
echo Starting ShareLink backend on http://localhost:8080
echo Test database: %DB_DSN%
echo.
go run cmd/server/main.go
exit /b %ERRORLEVEL%

:frontend
cd /d "%ROOT%frontend"
set "VITE_API_BASE_URL=http://localhost:8080"
echo Starting ShareLink frontend on http://localhost:5173/admin
echo.
npm run dev -- --host 127.0.0.1
exit /b %ERRORLEVEL%
