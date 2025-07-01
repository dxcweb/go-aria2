package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/dxcweb/go-aria2/aria2"
)

func main() {
	// 示例URL
	url := "https://repo.anaconda.com/miniconda/Miniconda3-py39_25.5.1-0-Windows-x86_64.exe"
	dir := ""

	// 记录开始时间
	startTime := time.Now()

	// 定义回调函数
	callback := func(status *aria2.DownloadStatus) {
		// 计算当前已过时间
		currentTime := time.Now()
		elapsed := currentTime.Sub(startTime)

		fmt.Printf("下载状态: %s (已用时: %v)\n", status.Status, elapsed)
		if status.TotalLength != "" {
			total, _ := strconv.ParseInt(status.TotalLength, 10, 64)
			completed, _ := strconv.ParseInt(status.CompletedLength, 10, 64)
			if total > 0 {
				progress := float64(completed) / float64(total) * 100
				fmt.Printf("进度: %.2f%% (%s/%s)\n", progress, status.CompletedLength, status.TotalLength)
			}
		}
		if status.DownloadSpeed != "" {
			// 将下载速度从字节转换为MB/s
			speedBytes, err := strconv.ParseInt(status.DownloadSpeed, 10, 64)
			if err == nil {
				speedMB := float64(speedBytes) / (1024 * 1024) // 转换为MB
				fmt.Printf("下载速度: %.2f MB/s\n", speedMB)
			} else {
				fmt.Printf("下载速度: %s/s\n", status.DownloadSpeed)
			}
		}
		if status.ErrorMessage != "" {
			fmt.Printf("错误信息: %s\n", status.ErrorMessage)
		}
		fmt.Println("---")
	}

	// 开始下载
	fmt.Println("开始下载...")
	path, err := aria2.Download(url, dir, callback)
	if err != nil {
		log.Fatalf("下载失败: %v", err)
	}
	fmt.Println("下载完成，路径为", path)
}
