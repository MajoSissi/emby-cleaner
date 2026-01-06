#!/bin/bash

# 本地构建脚本
# 用于在本地编译所有平台的可执行文件

set -e

echo "=== Emby Auto Cleaner 本地构建 ==="

# 创建build目录
mkdir -p build

# 定义构建配置
declare -A PLATFORMS=(
  ["linux/amd64"]="emby-auto-cleaner"
  ["windows/amd64"]="emby-auto-cleaner.exe"
)

# 构建每个平台
for GOOS_GOARCH in "${!PLATFORMS[@]}"; do
  OUTPUT=${PLATFORMS[$GOOS_GOARCH]}
  GOOS=${GOOS_GOARCH%/*}
  GOARCH=${GOOS_GOARCH#*/}

  echo ""
  echo "构建: $GOOS/$GOARCH -> $OUTPUT"

  env GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=0 go build \
    -ldflags="-s -w" \
    -o "build/$OUTPUT" \
    .

  if [ $? -eq 0 ]; then
    echo "✓ 构建成功: $OUTPUT"
  else
    echo "✗ 构建失败: $OUTPUT"
    exit 1
  fi
done

# 复制配置文件到build目录
echo ""
echo "复制配置文件..."
cp emby-auto-cleaner.yaml build/

echo ""
echo "=== 所有构建完成 ==="
ls -lh build/
