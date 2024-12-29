#!/bin/bash
echo "Starting build..."
rm -f automated-ticket-booker.exe
echo "Removed old build"
sleep 1
cd "cmd/main"
echo "Changed directory to cmd/main"
sleep 1

cp "../../versioninfo.json" "." 2>/dev/null
echo "Copied versioninfo.json"
sleep 1

cp "../../static/rail.ico" "." 2>/dev/null
echo "Copied rail.ico"
sleep 1

goversioninfo -platform-specific=true
echo "Ran goversioninfo"
sleep 1

go build
echo "Build complete"
sleep 1

echo "Cleaning up files..."
rm -f versioninfo.json
rm -f rail.ico
rm -f *.syso
mv main ../../automated-ticket-booker 2>/dev/null
echo "Naming and Moving executable to root directory"
sleep 1

cd "../.."
echo "Done!"
exit 0