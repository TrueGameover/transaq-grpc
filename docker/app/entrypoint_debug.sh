#!/usr/bin/env bash

/usr/bin/wine64 ~/go/bin/windows_amd64/dlv.exe --listen=:2345 --headless=true --api-version=2 --accept-multiclient exec /app/bin/server.exe
