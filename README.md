# Emby Cleaner

可以自动删除已观看的剧集文件。

## 功能特性

- 删除已观看超过指定天数的剧集文件
- 保留每个剧集最近N集不删除
- 支持只清理特定媒体库
- 支持只清理特定标签的文件
- 白名单保护（收藏和指定标签的文件不会被删除）
- 自动清理空文件夹

## 使用方法

1. 使用配置文件：

```bash
# 编辑配置文件，填写你的Emby服务器地址、用户名和密码
vim emby-cleaner.yaml  # 或使用其他编辑器
```

2. 运行程序：

```bash
go run main.go
# 或使用指定配置文件
go run main.go /path/to/config.yaml
```

3. 编译可执行文件：

```bash
# Linux/Mac
./build.sh

# Windows
build.bat

# 或手动编译
go build -o build/emby-cleaner .
```

## 配置说明

### Emby服务配置

- `url`: Emby服务器地址
- `username`: Emby用户名
- `password`: Emby密码

### 清理规则配置

- `watched_days_ago`: 删除多少天前观看的剧集
- `keep_latest_episodes`: 每个剧集保留最近的多少集
- `library_names`: 只清理指定的媒体库（留空表示全部）
- `tag_filters`: 只清理包含特定标签的剧集（留空表示不筛选）
- `protect_tags`: 包含这些标签的剧集不会被删除
- `protect_favorites`: 是否保护收藏的剧集
- `dry_run`: 是否为模拟运行（true只显示，不实际删除）
- `remove_empty_folders`: 是否删除空文件夹

## 安全建议

首次使用时建议将 `dry_run` 设置为 `true`，查看将要删除的文件列表，确认无误后再设置为 `false` 进行实际删除。
