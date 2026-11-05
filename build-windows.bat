@echo off
setlocal

echo ============================================
echo   Photon - Windows Production Build
echo ============================================
echo.

REM Make sure wails3 CLI is available
where wails3 >nul 2>nul
if errorlevel 1 (
    echo [ERROR] wails3 CLI not found in PATH.
    echo Install it with: go install github.com/wailsapp/wails/v3/cmd/wails3@latest
    pause
    exit /b 1
)

REM Make sure Go is available
where go >nul 2>nul
if errorlevel 1 (
    echo [ERROR] Go not found in PATH. Install it from https://go.dev/dl/
    pause
    exit /b 1
)

REM Make sure Node/npm is available for the frontend build
where npm >nul 2>nul
if errorlevel 1 (
    echo [ERROR] npm not found in PATH. Install Node.js from https://nodejs.org/
    pause
    exit /b 1
)

REM makensis is needed for the "package" step (NSIS installer)
where makensis >nul 2>nul
if errorlevel 1 (
    echo [WARNING] makensis not found in PATH.
    echo The build will succeed but packaging the installer will fail.
    echo Install NSIS from https://nsis.sourceforge.io/ and add it to PATH.
    echo.
)

echo Running: wails3 task windows:production
echo.
wails3 task windows:production

if errorlevel 1 (
    echo.
    echo [ERROR] Build failed.
    pause
    exit /b 1
)

echo.
echo ============================================
echo   Build finished successfully.
echo   Check the bin\ folder for the output.
echo ============================================
pause
