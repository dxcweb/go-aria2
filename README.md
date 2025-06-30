# go-aria2

一个基于 Go 语言开发的 Aria2 下载器封装库，提供了简单易用的 API 来管理文件下载任务。该库使用 Go 1.16+ 的 `embed` 功能内置了跨平台的 Aria2c 二进制文件，支持 Windows、Linux 和 macOS 系统。

## 🚀 功能特性

- **跨平台支持**: 支持 Windows、Linux 和 macOS 系统
- **内置二进制文件**: 使用 Go embed 功能自动嵌入对应平台的 Aria2c 二进制文件，无需额外安装
- **简单易用的 API**: 提供简洁的 Go API 接口
- **实时下载状态**: 支持下载进度回调，实时获取下载状态
- **断点续传**: 支持下载中断后继续下载
- **多线程下载**: 支持多连接并发下载，提高下载速度
- **自动端口管理**: 自动寻找可用端口启动 RPC 服务

## 🔧 Go Embed 功能

本库充分利用了 Go 1.16+ 引入的 `embed` 功能，将 Aria2c 二进制文件直接嵌入到 Go 程序中：

### 嵌入实现

```go
import _ "embed"

// 嵌入不同平台的Aria2c二进制文件
//go:embed binaries/aria2c.exe
var aria2cWindows []byte

//go:embed binaries/aria2c-linux
var aria2cLinux []byte

//go:embed binaries/aria2c-darwin
var aria2cDarwin []byte
```

### 优势

- **零依赖**: 无需用户手动下载或安装 Aria2c
- **版本控制**: 二进制文件版本与代码版本保持一致
- **部署简单**: 单个可执行文件包含所有依赖
- **跨平台**: 自动根据运行平台选择正确的二进制文件

### 自动提取

库会在首次使用时自动将嵌入的二进制文件提取到系统对应的应用数据目录：

- **Windows**: `%LOCALAPPDATA%\aria2`
- **macOS**: `~/Library/Application Support/aria2`
- **Linux**: `~/.local/share/aria2`

## 📦 安装

```bash
go get github.com/dxcweb/go-aria2
```

## 🛠️ 快速开始

### 基本使用

```go
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/dxcweb/go-aria2/aria2"
)

func main() {
	// 定义下载回调函数
	callback := func(status *aria2.DownloadStatus) {
		fmt.Printf("下载状态: %s\n", status.Status)
		if status.TotalLength != "" {
			fmt.Printf("进度: %s/%s\n", status.CompletedLength, status.TotalLength)
		}
		if status.DownloadSpeed != "" {
			fmt.Printf("下载速度: %s/s\n", status.DownloadSpeed)
		}
	}

	// 开始下载
	url := "https://repo.anaconda.com/miniconda/Miniconda3-py39_25.5.1-0-Windows-x86_64.exe"
	err := aria2.Download(url, "", "", callback)
	if err != nil {
		log.Fatalf("下载失败: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
}

```

## 📚 API 文档

### 主要类型

#### DownloadStatus
下载状态结构体，包含以下字段：

```go
type DownloadStatus struct {
    GID             string // 下载任务的GID
    Status          string // 状态：active, waiting, paused, error, complete, removed
    TotalLength     string // 文件总大小
    CompletedLength string // 已完成大小
    DownloadSpeed   string // 下载速度
    PieceLength     string // 分片大小
    NumPieces       string // 分片数量
    Connections     string // 连接数
    ErrorCode       string // 错误代码
    ErrorMessage    string // 错误信息
    Dir             string // 下载目录
}
```
## 🔧 配置选项

Aria2c 启动时会使用以下默认配置：

- **RPC 端口**: 自动寻找可用端口（默认从 6800 开始）
- **磁盘缓存**: 64MB
- **断点续传**: 启用
- **最大连接数**: 每服务器 16 个连接
- **单任务连接数**: 64 个
- **最小分片大小**: 1MB
- **优化并发下载**: 启用

## 📁 项目结构

```
go-aria2/
├── aria2/
│   ├── aria2.go          # 主要功能实现
│   ├── embedder.go       # Go embed 功能实现，二进制文件嵌入和提取
│   └── binaries/         # 跨平台二进制文件（通过 embed 嵌入）
│       ├── aria2c.exe    # Windows 版本
│       ├── aria2c-linux  # Linux 版本
│       └── aria2c-darwin # macOS 版本
├── example.go            # 使用示例
├── go.mod               # Go 模块文件
└── README.md            # 项目文档
```

### Embed 文件说明

- `embedder.go`: 使用 Go 1.16+ 的 `embed` 功能将二进制文件嵌入到程序中
- `binaries/`: 包含各平台的 Aria2c 二进制文件，通过 `//go:embed` 指令嵌入
- 运行时自动提取: 程序首次运行时会自动将二进制文件提取到系统应用数据目录

## 🌍 跨平台支持

该库使用 Go embed 功能支持以下平台：

- **Windows**: 使用嵌入的 `aria2c.exe`
- **Linux**: 使用嵌入的 `aria2c-linux`
- **macOS**: 使用嵌入的 `aria2c-darwin`

### Embed 优势

- **无需外部依赖**: 所有平台的二进制文件都已嵌入到程序中
- **自动平台检测**: 运行时自动检测当前平台并使用对应的二进制文件
- **统一部署**: 同一个程序可以在不同平台上运行，无需分别编译
- **版本一致性**: 确保二进制文件版本与库版本完全匹配

二进制文件会在首次使用时自动提取到系统对应的应用数据目录：
- Windows: `%LOCALAPPDATA%\aria2`
- macOS: `~/Library/Application Support/aria2`
- Linux: `~/.local/share/aria2`

## 🤝 贡献

欢迎提交 Issue 和 Pull Request 来改进这个项目。

## 📄 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。

## 🙏 致谢

- [Aria2](https://aria2.github.io/) - 强大的多协议下载工具
- Go 语言社区 - 提供了优秀的开发工具和生态系统