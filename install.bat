@echo off
:: install.bat — double-clickable Windows installer for non-PowerShell users.
:: Bootstraps install.ps1 from GitHub. Safe to run twice.

setlocal ENABLEDELAYEDEXPANSION

set "REPO=Prince2412k2/logify"
if "%LOGIFY_REPO%" NEQ "" set "REPO=%LOGIFY_REPO%"

echo.
echo   logify installer (Windows)
echo   --------------------------
echo   repo:    %REPO%
if "%LOGIFY_VERSION%" NEQ "" echo   version: %LOGIFY_VERSION%
echo.

where powershell >/dev/null 2>&1
if errorlevel 1 (
  echo [ERROR] PowerShell not found. Use Windows Terminal or PowerShell 5+.
  pause
  exit /b 1
)

powershell -NoProfile -ExecutionPolicy Bypass -Command ^
  "$env:LOGIFY_REPO='%REPO%'; $env:LOGIFY_VERSION='%LOGIFY_VERSION%'; iwr -useb https://raw.githubusercontent.com/%REPO%/main/install.ps1 | iex"

if errorlevel 1 (
  echo.
  echo [ERROR] install.ps1 reported failure
  pause
  exit /b 1
)

echo.
echo Done. Open a new terminal and run: logify version
pause
endlocal
