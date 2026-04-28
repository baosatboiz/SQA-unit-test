@echo off
setlocal enabledelayedexpansion
echo ====================================
echo Cinema Postman API Test Suite
echo ====================================

REM Pre-check API gateway
curl -s -o nul -w "%%{http_code}" http://localhost:8000/health > tmp_status.txt
set /p STATUS=<tmp_status.txt
del tmp_status.txt
if not "%STATUS%"=="200" (
    echo [ERROR] cinema-project API gateway not reachable at http://localhost:8000/health
    echo        Expected HTTP 200, got: %STATUS%
    echo        Start the backend first, then re-run this script.
    exit /b 1
)
echo [OK] API gateway is up.

REM Timestamp for report filename
for /f "tokens=2 delims==" %%a in ('wmic OS Get localdatetime /value') do set "DT=%%a"
set "TS=%DT:~0,8%-%DT:~8,6%"

REM Ensure reports directory exists
if not exist reports mkdir reports

REM Run Newman
echo Running Newman...
call newman run collections\Cinema-API.postman_collection.json ^
  -e environments\Local.postman_environment.json ^
  -r htmlextra,cli ^
  --reporter-htmlextra-export reports\newman-report-%TS%.html ^
  --reporter-htmlextra-title "Cinema API Test Report" ^
  --reporter-htmlextra-darkTheme ^
  --timeout-request 10000
set RC=%ERRORLEVEL%

echo.
echo ====================================
echo Newman exit code: %RC%
echo ====================================

REM Cleanup DB
echo Running DB cleanup...
set PGPASSWORD=Trang@051203
psql -h localhost -p 5433 -U postgres -d cinema_app -f cleanup.sql
if %ERRORLEVEL% neq 0 (
    echo [WARN] DB cleanup failed or psql not in PATH — run cleanup.sql manually.
)
set PGPASSWORD=

REM Open report in browser
if exist "reports\newman-report-%TS%.html" (
    echo Opening report: reports\newman-report-%TS%.html
    start "" "reports\newman-report-%TS%.html"
) else (
    echo [WARN] HTML report not found. Is newman-reporter-htmlextra installed?
    echo        Run: npm install -g newman-reporter-htmlextra
)

exit /b %RC%
