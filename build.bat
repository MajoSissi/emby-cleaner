@echo off
REM Windows构建脚本
REM 用于在Windows本地编译所有平台的可执行文件

echo === Emby Auto Cleaner 本地构建 ===

REM 创建build目录
if not exist build mkdir build

REM 构建Windows版本
echo.
echo 构建: windows/amd64
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -ldflags="-s -w" -o build/emby-auto-cleaner.exe .
if %ERRORLEVEL% EQU 0 (
    echo ✓ 构建成功: emby-auto-cleaner.exe
) else (
    echo ✗ 构建失败
    exit /b 1
)

REM 构建Linux版本
echo.
echo 构建: linux/amd64
set GOOS=linux
set GOARCH=amd64
go build -ldflags="-s -w" -o build/emby-auto-cleaner .
if %ERRORLEVEL% EQU 0 (
    echo ✓ 构建成功: emby-auto-cleaner
) else (
    echo ✗ 构建失败
    exit /b 1
)

REM 复制配置文件到build目录
echo.
echo 复制配置文件...
copy emby-auto-cleaner.yaml build\

echo.
echo === 所有构建完成 ===
dir build\
