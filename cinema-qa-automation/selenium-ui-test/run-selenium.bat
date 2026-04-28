@echo off
setlocal enabledelayedexpansion
echo ====================================
echo Cinema Selenium UI Test Suite
echo ====================================

REM Pre-check API gateway up
curl -s -o nul -w "%%{http_code}" http://localhost:8000/health > tmp_status.txt
set /p STATUS=<tmp_status.txt
del tmp_status.txt
if not "%STATUS%"=="200" (
    echo [ERROR] cinema-project API gateway not reachable at http://localhost:8000/health (status=%STATUS%)
    echo Please start: cd ../../cinema-project ^&^& docker compose up -d
    exit /b 1
)
echo [OK] API gateway is up.

REM Run tests
call mvn clean test -DsuiteXmlFile=testng.xml
set RC=%ERRORLEVEL%

REM Open latest report
for /f "delims=" %%f in ('dir /b /od reports\extent-report-*.html 2^>nul') do set LATEST=%%f
if defined LATEST (
    echo Opening report: reports\!LATEST!
    start "" "reports\!LATEST!"
)

exit /b %RC%
