@echo off
setlocal ENABLEDELAYEDEXPANSION
set "REPO=Prince2412k2/logify"
if "%LOGIFY_REPO%" NEQ "" set "REPO=%LOGIFY_REPO%"
echo.
echo  logify installer (Windows)
echo.
where powershell >/dev/null 2>&1 || (echo PowerShell not found & pause & exit /b 1)
powershell -NoProfile -ExecutionPolicy Bypass -Command "$env:LOGIFY_REPO='%REPO%'; $env:LOGIFY_VERSION='%LOGIFY_VERSION%'; iwr -useb https://raw.githubusercontent.com/%REPO%/main/install.ps1 | iex"
if errorlevel 1 (pause & exit /b 1)
echo.
echo Done. Run: logify version
pause
endlocal
