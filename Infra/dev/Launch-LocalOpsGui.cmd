@echo off
setlocal
set "REPO_ROOT=%~dp0..\.."
set "AMANDACORE_REPO_ROOT=%REPO_ROOT%"
set "LOCAL_CONTROLS_PROJECT=%REPO_ROOT%\Client\Tools\AmandaCore.LocalControls\AmandaCore.LocalControls.csproj"
set "LOCAL_CONTROLS_EXE=%REPO_ROOT%\Client\Tools\AmandaCore.LocalControls\bin\Debug\net8.0-windows\AmandaCore.LocalControls.exe"
if exist "%LOCAL_CONTROLS_PROJECT%" (
    dotnet build "%LOCAL_CONTROLS_PROJECT%"
    if errorlevel 1 exit /b %ERRORLEVEL%
)
if not exist "%LOCAL_CONTROLS_EXE%" (
    echo Local Controls executable was not found: "%LOCAL_CONTROLS_EXE%"
    exit /b 1
)
start "" "%LOCAL_CONTROLS_EXE%"
