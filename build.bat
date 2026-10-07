@echo off
echo Building Four-PlayerArmyChess...
go build -ldflags="-H windowsgui" -o Four-PlayerArmyChess.exe
if %errorlevel% equ 0 (
    echo Build OK. Launching...
    start Four-PlayerArmyChess.exe
) else (
    echo Build failed. Check the output above.
)
pause
