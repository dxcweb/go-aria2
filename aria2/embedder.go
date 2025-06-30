package aria2

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// 嵌入不同平台的Aria2c二进制文件
// 这里需要预先下载对应平台的aria2c二进制文件并放置在binaries目录中

//go:embed binaries/aria2c.exe
var aria2cWindows []byte

//go:embed binaries/aria2c-linux
var aria2cLinux []byte

//go:embed binaries/aria2c-darwin
var aria2cDarwin []byte

// GetEmbeddedBinaryData 根据当前平台返回对应的二进制文件数据
func GetEmbeddedBinaryData() ([]byte, error) {
	switch runtime.GOOS {
	case "windows":
		return aria2cWindows, nil
	case "linux":
		return aria2cLinux, nil
	case "darwin":
		return aria2cDarwin, nil
	default:
		return nil, fmt.Errorf("不支持的平台: %s", runtime.GOOS)
	}
}

// GetEmbeddedBinaryName 根据当前平台返回对应的二进制文件名
func GetEmbeddedBinaryName() (string, error) {
	switch runtime.GOOS {
	case "windows":
		return "aria2c.exe", nil
	case "linux":
		return "aria2c", nil
	case "darwin":
		return "aria2c", nil
	default:
		return "", fmt.Errorf("不支持的平台: %s", runtime.GOOS)
	}
}

// ExtractBinary 将嵌入的二进制文件提取到app目录
func ExtractBinary() (string, error) {
	filename, err := GetEmbeddedBinaryName()
	if err != nil {
		return "", err
	}

	// 获取跨平台的应用数据目录
	appDir, err := getAppDataDir()
	if err != nil {
		return "", fmt.Errorf("无法获取应用程序数据目录: %w", err)
	}

	// 构建二进制文件路径
	binaryPath := filepath.Join(appDir, filename)

	// 检查文件是否已存在
	if _, err := os.Stat(binaryPath); err == nil {
		// 文件已存在，直接返回路径
		return binaryPath, nil
	}
	if err := CheckBinaryExists(); err != nil {
		return "", err
	}

	err = os.MkdirAll(appDir, 0755)
	if err != nil {
		return "", fmt.Errorf("创建应用程序目录失败: %w", err)
	}

	data, err := GetEmbeddedBinaryData()
	if err != nil {
		return "", fmt.Errorf("无法获取嵌入的二进制文件数据: %w", err)
	}

	// 写入二进制文件
	err = os.WriteFile(binaryPath, data, 0755)
	if err != nil {
		return "", fmt.Errorf("写入二进制文件失败: %w", err)
	}

	return binaryPath, nil
}

// CheckBinaryExists 检查二进制文件是否存在
func CheckBinaryExists() error {
	data, err := GetEmbeddedBinaryData()
	if err != nil {
		return err
	}
	// 检查是否为占位文件
	if len(data) <= 2 {
		return fmt.Errorf("未找到 aria2c 二进制文件 - 请先运行下载脚本")
	}

	return nil
}

// getAppDataDir 获取跨平台的应用数据目录
func getAppDataDir() (string, error) {
	var baseDir string

	switch runtime.GOOS {
	case "windows":
		// Windows: %LOCALAPPDATA%\aria2c-go
		baseDir = os.Getenv("LOCALAPPDATA")
		if baseDir == "" {
			// 如果环境变量不存在，使用用户主目录
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			baseDir = filepath.Join(homeDir, "AppData", "Local")
		}
	case "darwin":
		// macOS: ~/Library/Application Support/aria2c-go
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		baseDir = filepath.Join(homeDir, "Library", "Application Support")
	case "linux":
		// Linux: ~/.local/share/aria2c-go
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		// 优先使用 XDG_DATA_HOME 环境变量
		xdgDataHome := os.Getenv("XDG_DATA_HOME")
		if xdgDataHome != "" {
			baseDir = xdgDataHome
		} else {
			baseDir = filepath.Join(homeDir, ".local", "share")
		}
	default:
		return "", fmt.Errorf("不支持的平台: %s", runtime.GOOS)
	}

	// 在基础目录下创建 aria2子目录
	appDir := filepath.Join(baseDir, "aria2")
	return appDir, nil
}
