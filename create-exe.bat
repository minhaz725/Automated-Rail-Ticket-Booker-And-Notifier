@echo off
echo Starting build...
timeout /t 1 /nobreak >nul

del automated-ticket-booker.exe
echo Removed old build
timeout /t 1 /nobreak >nul

cd "cmd\main"
echo Changed directory to cmd\main
timeout /t 1 /nobreak >nul

copy "..\..\versioninfo.json" ".">nul
echo Copied versioninfo.json
timeout /t 1 /nobreak >nul

copy "..\..\static\rail.ico" ".">nul
echo Copied rail.ico
timeout /t 1 /nobreak >nul

goversioninfo -platform-specific=true
echo Ran goversioninfo
timeout /t 1 /nobreak >nul

go build
echo Build complete
timeout /t 1 /nobreak >nul

echo Cleaning up files...
del versioninfo.json
del rail.ico
del *.syso

move main.exe ..\..\automated-ticket-booker.exe>nul
echo Naming and Moving executable to root directory
timeout /t 1 /nobreak >nul
cd "..\.."
echo Done!
exit /b 0