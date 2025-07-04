# Easily Minecraft Manager (EMCM)

![EMCM](https://socialify.git.ci/Easily-miku/EMCM/image?font=Raleway&forks=1&language=1&logo=https%3A%2F%2Fimg.picui.cn%2Ffree%2F2025%2F07%2F04%2F6867c3c7f243f.png&name=1&owner=1&pattern=Circuit+Board&stargazers=1&theme=Auto)
**简化 Minecraft 服务器管理 - 让开服变得轻松愉快**

[![GitHub release](https://img.shields.io/github/release/Easily-Miku/EMCM.svg)](https://github.com/Easily-Miku/EMCM/releases)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/Easily-Miku/EMCM)](https://goreportcard.com/report/github.com/Easily-Miku/EMCM)

EMCM 是一个轻量级命令行工具，帮助您轻松管理 Minecraft 服务器。通过集成无极镜像，您可以快速下载各种服务端核心（Paper、Forge、Arclight 等），并提供了直观的菜单系统和日志翻译功能。

## ✨ 功能亮点

- ⚡ **一键下载服务端**：从无极镜像获取最新服务端核心
- 🌐 **跨平台支持**：完美兼容 Windows、Linux、macOS
- 📜 **实时日志翻译**：中文显示 Minecraft 服务器日志
- ☕ **智能 Java 管理**：自动检测并推荐 Java 版本
- 🚀 **多服务器支持**：同时管理最多 10 个服务器实例
- ⚙️ **自定义启动参数**：灵活配置 JVM 启动选项
- 📦 **轻量高效**：单文件程序，无需额外依赖
- 🎨 **彩色界面**：直观的彩色菜单和状态提示

## 📥 安装

### 预编译版本

前往 [Releases 页面](https://github.com/Easily-Miku/EMCM/releases) 下载对应平台的二进制文件：

| 平台              | 文件名称                     |
|-------------------|-----------------------------|
| Windows (64-bit)  | `emcm-windows-amd64.exe`    |
| Linux (64-bit)    | `emcm-linux-amd64`          |
| macOS (Intel)     | `emcm-macos-amd64`          |
| macOS (Apple Silicon)| `emcm-macos-arm64`        |

### 从源码编译

1. 确保已安装 Go 1.16+
2. 克隆仓库：
   ```bash
   git clone https://github.com/Easily-Miku/EMCM.git
   cd emcm
   ```
3. 安装依赖：
   ```bash
   go get github.com/common-nighthawk/go-figure
   ```
4. 编译：
   ```bash
   # 编译当前平台
   go build -o emcm
   
   # 编译 Windows 版本
   env GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o emcm.exe
   
   # 编译 Linux 版本
   env GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o emcm-linux
   
   # 编译 macOS 版本
   env GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o emcm-macos
   ```

## 🚀 快速开始

### 首次运行

1. 启动 EMCM：
   ```bash
   # Windows
   emcm.exe
   
   # Linux/macOS
   ./emcm
   ```
2. 程序将引导您创建第一个服务器实例
3. 选择服务端类型和版本
4. 自动下载服务端核心文件

### 基本命令

```bash
# 列出可用服务端
emcm list

# 查看服务端支持的 MC 版本
emcm versions Paper

# 下载 Paper 1.20.1 最新版
emcm download Paper 1.20.1

# 启动服务器
emcm start server-1

# 停止服务器
emcm stop server-1
```

## 📖 核心功能

### 服务器管理
- 创建、重命名和删除服务器实例
- 最多支持 10 个服务器实例
- 同时运行多个服务器
- 实时查看服务器日志

### Java 环境管理
- 自动检测系统 Java 安装
- 支持添加多个 Java 版本
- 为不同服务器配置专属 Java 环境
- 根据 MC 版本智能推荐 Java 版本

### 日志翻译
- 内置基础日志翻译规则
- 支持自定义翻译字典
- 实时翻译服务器日志
- 可恢复默认字典

### 高级配置
- 自定义 JVM 启动参数
- 设置默认内存分配
- 配置服务端启动选项
- 管理多个 Java 版本

## 📚 使用指南

### 主菜单
```
███████╗███╗   ███╗ ██████╗███╗   ███╗
██╔════╝████╗ ████║██╔════╝████╗ ████║
█████╗  ██╔████╔██║██║     ██╔████╔██║
██╔══╝  ██║╚██╔╝██║██║     ██║╚██╔╝██║
███████╗██║ ╚═╝ ██║╚██████╗██║ ╚═╝ ██║
╚══════╝╚═╝     ╚═╝ ╚═════╝╚═╝     ╚═╝
                                      
Easily Minecraft Manager v2.1
Author: Easily-Miku
GitHub: https://github.com/Easily-miku
--------------------------------------

1. 启动服务器
2. 停止服务器
3. 管理服务器实例
4. 下载服务端核心
5. Java环境管理
6. 内存设置
7. 编辑日志翻译字典
8. 退出
--------------------------------------
请选择操作: 
```

### 创建服务器实例
1. 输入服务器名称
2. 选择创建方式：
   - 从无极镜像下载新服务端
   - 使用现有服务端文件
3. 选择服务端类型（Paper、Forge等）
4. 选择 MC 版本
5. 选择构建版本
6. 自动配置 Java 环境

### 管理服务器实例
- **重命名实例**：修改服务器显示名称
- **配置Java环境**：为服务器指定 Java 路径
- **配置启动参数**：自定义 JVM 启动选项
- **删除实例**：移除不再需要的服务器

## 🛠 技术细节

### 文件结构
```
.emcm/
├── servers/              # 服务器实例
│   └── Paper-1.20.1/
│       ├── server.jar    # 服务端核心
│       ├── server.properties
│       └── eula.txt
├── cache/                # API缓存
├── logs.dict             # 日志翻译字典
└── emcm.config           # EMCM配置文件
```

### 日志翻译字典格式
```
原始日志正则表达式#翻译文本
```
示例：
```
Player [a-zA-Z0-9_]+ joined#玩家 $0 加入游戏
Done \(\d+\.\d+s\)!#启动完成 (耗时 $0 秒)
```

## 🤝 贡献指南

欢迎贡献！请遵循以下步骤：

1. Fork 项目仓库
2. 创建新分支 (`git checkout -b feature/awesome-feature`)
3. 提交更改 (`git commit -m 'Add awesome feature'`)
4. 推送到分支 (`git push origin feature/awesome-feature`)
5. 创建 Pull Request

## ❓ 常见问题

### Windows 下无法运行？
- 确保下载的是 Windows 版本的可执行文件
- 在 PowerShell 或命令提示符中运行
- 尝试静态编译版本

### 日志翻译不工作？
- 检查 `.emcm/logs.dict` 文件是否存在
- 确保字典文件格式正确
- 尝试恢复默认字典

### 如何添加自定义 Java 版本？
1. 在主菜单中选择 "Java环境管理"
2. 选择 "添加Java版本"
3. 输入 Java 版本号（如 17）
4. 输入 Java 完整路径

## 📜 许可证

本项目采用 [MIT 许可证](LICENSE)

---
**EMCM © 2025 Easily-Miku**  
让 Minecraft 服务器管理变得简单！  
GitHub: [https://github.com/Easily-Miku/EMCM](https://github.com/Easily-Miku/EMCM)
