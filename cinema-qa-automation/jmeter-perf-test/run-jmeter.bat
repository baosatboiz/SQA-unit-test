@echo off
setlocal enabledelayedexpansion
echo ====================================
echo Cinema JMeter Performance Test
echo ====================================

REM Pre-check
curl -s -o nul -w "%%{http_code}" http://localhost:8000/health > tmp_status.txt
set /p STATUS=<tmp_status.txt
del tmp_status.txt
if not "%STATUS%"=="200" (
    echo [ERROR] cinema-project API gateway not reachable.
    exit /b 1
)
echo [OK] API gateway is up.

REM Timestamp
for /f "tokens=2 delims==" %%a in ('wmic OS Get localdatetime /value') do set "DT=%%a"
set "TS=%DT:~0,8%-%DT:~8,6%"

if not exist reports mkdir reports
if exist reports\results.jtl del reports\results.jtl

REM Run JMeter non-GUI
call jmeter -n -t test-plans\cinema-user-journey.jmx ^
  -l reports\results.jtl ^
  -e -o reports\dashboard-%TS%
set RC=%ERRORLEVEL%

REM Cleanup DB
echo Running cleanup...
set PGPASSWORD=Trang@051203
psql -h localhost -p 5433 -U postgres -d cinema_app -f cleanup.sql
set PGPASSWORD=

REM Open dashboard
start "" "reports\dashboard-%TS%\index.html"

exit /b %RC%
