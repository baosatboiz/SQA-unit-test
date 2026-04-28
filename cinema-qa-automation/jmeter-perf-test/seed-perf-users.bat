@echo off
setlocal enabledelayedexpansion
echo Seeding 100 perf users via API...

REM Pre-check
curl -s -o nul -w "%%{http_code}" http://localhost:8000/health > tmp_status.txt
set /p STATUS=<tmp_status.txt
del tmp_status.txt
if not "%STATUS%"=="200" (
    echo [ERROR] cinema-project API gateway not reachable. Start cinema-project first.
    exit /b 1
)

set OK=0
set FAIL=0
for /l %%i in (1,1,100) do (
    set "N=00%%i"
    set "N=!N:~-3!"
    curl -s -o nul -w "%%{http_code}" -X POST http://localhost:8000/api/v1/auth/register ^
      -H "Content-Type: application/json" ^
      -d "{\"email\":\"qa_perf_user_!N!@test.com\",\"password\":\"Pass@1234\",\"confirmPassword\":\"Pass@1234\",\"firstName\":\"Perf\",\"lastName\":\"User!N!\",\"address\":\"Address !N!\"}" > tmp_code.txt
    set /p CODE=<tmp_code.txt
    if "!CODE!"=="201" (set /a OK+=1) else (set /a FAIL+=1)
)
del tmp_code.txt 2>nul
echo Done. OK=!OK!, FAIL=!FAIL! (FAIL count includes duplicates from prior runs - that's OK)
echo Verify with: psql -h localhost -p 5433 -U postgres -d cinema_app -c "SELECT COUNT(*) FROM users WHERE email LIKE 'qa_perf_%%'"
