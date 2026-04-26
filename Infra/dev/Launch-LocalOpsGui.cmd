@echo off
setlocal
set "REPO_ROOT=%~dp0..\.."
set "AMANDACORE_REPO_ROOT=%REPO_ROOT%"
set "LOCAL_CONTROLS_PROJECT=%REPO_ROOT%\Client\Tools\AmandaCore.LocalControls\AmandaCore.LocalControls.csproj"
set "LOCAL_CONTROLS_EXE=%REPO_ROOT%\Client\Tools\AmandaCore.LocalControls\bin\Debug\net8.0-windows\AmandaCore.LocalControls.exe"
dotnet build "%LOCAL_CONTROLS_PROJECT%"
if errorlevel 1 exit /b %ERRORLEVEL%
start "" "%LOCAL_CONTROLS_EXE%"
