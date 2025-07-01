package aria2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// DownloadStatus 下载状态结构体
type DownloadStatus struct {
	GID             string `json:"gid"`             // 下载任务的GID
	Status          string `json:"status"`          // 状态：active, waiting, paused, error, complete, removed
	TotalLength     string `json:"totalLength"`     // 文件总大小
	CompletedLength string `json:"completedLength"` // 已完成大小
	DownloadSpeed   string `json:"downloadSpeed"`   // 下载速度
	PieceLength     string `json:"pieceLength"`     // 分片大小
	NumPieces       string `json:"numPieces"`       // 分片数量
	Connections     string `json:"connections"`     // 连接数
	ErrorCode       string `json:"errorCode"`       // 错误代码
	ErrorMessage    string `json:"errorMessage"`    // 错误信息
	Files           []File `json:"files"`           // 文件列表
}
type File struct {
	Path string `json:"path"`
}

// URI URI信息结构体
type URI struct {
	URI    string `json:"uri"`
	Status string `json:"status"`
}

// Bittorrent BitTorrent信息结构体
type Bittorrent struct {
	Info *Info `json:"info"`
}

// Info 信息结构体
type Info struct {
	Name string `json:"name"`
}

// DownloadCallback 下载回调函数类型
type DownloadCallback func(status *DownloadStatus)

// DownloadResult 下载结果结构体
type DownloadResult struct {
	Status *DownloadStatus
	Error  error
}

type Aria2 struct {
	port       int
	mu         sync.Mutex
	running    bool
	cmd        *exec.Cmd
	ctx        context.Context
	cancel     context.CancelFunc
	httpClient *http.Client
}

// 全局实例
var aria2 = newDaemon()

// Download 包级别的下载函数，可以直接调用
func Download(url string, dir string, out string, callback DownloadCallback) (string, error) {
	if !aria2.IsRunning() {
		if err := aria2.Start(); err != nil {
			return "", err
		}
	}
	gid, err := aria2.AddUri(url, dir, out)
	if err != nil {
		return "", err
	}

	return aria2.monitorDownload(gid, callback)
}
func Stop() {
	aria2.Stop()
}

func newDaemon() *Aria2 {
	ctx, cancel := context.WithCancel(context.Background())

	return &Aria2{
		port:   findAvailablePort(6800),
		ctx:    ctx,
		cancel: cancel,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// DownloadFile 下载文件的便捷方法
func (a *Aria2) Download(url string) (string, error) {
	if !a.IsRunning() {
		return "", fmt.Errorf("aria2c没有运行")
	}
	return "", nil
}

// IsRunning 检查服务是否正在运行
func (a *Aria2) IsRunning() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.running
}

func (a *Aria2) Start() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	println("启动Aria2c")
	if a.running {
		return fmt.Errorf("aria2c已经运行")
	}

	binaryPath, err := ExtractBinary()
	if err != nil {
		return err
	}
	args := a.buildArgs()
	ctx, cancel := context.WithCancel(context.Background())
	a.ctx = ctx
	a.cancel = cancel

	a.cmd = exec.CommandContext(a.ctx, binaryPath, args...)

	// 在 Windows 上隐藏控制台窗口
	if a.cmd.SysProcAttr == nil {
		a.cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	a.cmd.SysProcAttr.HideWindow = true

	if err := a.cmd.Start(); err != nil {
		return fmt.Errorf("aria2c 进程启动失败: %v", err)
	}

	// 等待RPC服务启动
	if err := a.waitForRPC(); err != nil {
		return fmt.Errorf("RPC service failed to start: %w", err)
	}

	a.running = true
	go a.monitor()
	// 启动进程监控
	// a.processMonitor = make(chan struct{})
	// go a.monitorProcess()

	return nil
}

// monitor 监控进程状态
func (a *Aria2) monitor() {
	if a.cmd != nil {
		a.cmd.Wait()
		a.Stop()
	}
}

// 修改 Stop 方法
func (a *Aria2) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.running = false
	if a.cmd != nil && a.cmd.Process != nil {
		if err := a.cmd.Process.Kill(); err != nil {
			println("3333", "failed to kill aria2c process: %w", err)
			return fmt.Errorf("failed to kill aria2c process: %w", err)
		}
	}

	return nil
}

func findAvailablePort(port int) int {
	// 尝试监听该端口
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		// 端口被占用
		return findAvailablePort(port + 1)
	}
	// 端口可用，立即关闭监听器
	listener.Close()
	return port
}

// buildArgs 构建命令行参数
func (a *Aria2) buildArgs() []string {
	args := []string{
		"--rpc-listen-port=" + strconv.Itoa(a.port),
		"--disk-cache=64M",             // 磁盘缓存 有足够的内存空闲情况下适当增加
		"--always-resume=false",        // 始终尝试断点续传，无法断点续传则终止下载，默认：true
		"--max-resume-failure-tries=0", // 值为 0 时所有 URI 不支持断点续传时才从头开始下载
		"--enable-rpc=true",            //
		"--rpc-listen-all=true",
		"--continue=true",
		"--max-connection-per-server=16", // 单服务器最大连接线程数,  默认:1
		"--min-split-size=1M",            //  文件最小分段大小
		"--split=64",                     // 单任务最大连接线程数
		"--optimize-concurrent-downloads=true",
		"--log-level=error",
		"--http-accept-gzip=true",                 // GZip 支持，默认:false
		"--content-disposition-default-utf8=true", //使用 UTF-8 处理 Content-Disposition ，默认:false
		"--check-certificate=false",               // 禁用SSL证书验证
	}

	return args
}

// waitForRPC 等待RPC服务启动
// 这个函数会持续检查 aria2c 的 RPC 服务是否已经启动并可以接受连接
func (a *Aria2) waitForRPC() error {
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			// 如果超过10秒超时时间，返回超时错误
			return fmt.Errorf("等待RPC服务超时")
		case <-ticker.C:
			// 每100毫秒执行一次：尝试连接到 aria2c 的 RPC 端口
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", a.port), time.Second)
			if err == nil {
				// 如果连接成功（err == nil），说明 RPC 服务已经启动
				// 立即关闭连接（因为我们只是测试连接，不需要保持连接）
				conn.Close()
				// 返回 nil 表示成功，函数结束
				return nil
			}
			// 如果连接失败，继续下一次循环（100毫秒后再次尝试）
		case <-a.ctx.Done():
			// 如果上下文被取消（比如程序被中断），返回上下文取消错误
			return fmt.Errorf("ctx上下文已取消")
		}
	}

}

type jsonRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      string        `json:"id"`
}

// JSONRPCResponse JSON-RPC 响应结构
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
	ID      string          `json:"id"`
}

// JSONRPCError JSON-RPC 错误结构
type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (a *Aria2) Call(method string, params []interface{}) (json.RawMessage, error) {
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}
	url := fmt.Sprintf("http://127.0.0.1:%d/jsonrpc", a.port)
	// 发送 HTTP 请求
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// 执行请求
	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析响应
	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查错误
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("JSON-RPC错误 %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

func (a *Aria2) AddUri(uri string, dir string, out string) (string, error) {
	result, err := a.Call("aria2.addUri", []interface{}{
		[]string{uri}, // 第一个参数：URL数组
		map[string]interface{}{ // 第二个参数：选项对象
			"dir": dir,
			"out": out,
		},
	})
	if err != nil {
		return "", err
	}
	var gid string
	if err := json.Unmarshal(result, &gid); err != nil {
		return "", fmt.Errorf("解析GID失败: %w", err)
	}
	return gid, nil
}

// TellStatus 获取下载任务状态
func (a *Aria2) TellStatus(gid string) (*DownloadStatus, error) {
	result, err := a.Call("aria2.tellStatus", []interface{}{gid})
	if err != nil {
		return nil, err
	}
	var status DownloadStatus
	if err := json.Unmarshal(result, &status); err != nil {
		return nil, fmt.Errorf("解析状态失败: %w", err)
	}
	return &status, nil
}

// monitorDownload 监控下载状态直到完成或出错（同步版本）
func (a *Aria2) monitorDownload(gid string, callback DownloadCallback) (string, error) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			status, err := a.TellStatus(gid)
			if err != nil {
				return "", err
			}

			// 调用回调函数
			if callback != nil {
				callback(status)
			}

			// 检查是否完成或出错
			switch status.Status {
			case "complete":
				return status.Files[0].Path, nil
			case "error":
				return "", fmt.Errorf("下载出错: %s", status.ErrorMessage)
			}
		case <-a.ctx.Done():
			return "", fmt.Errorf("ctx上下文已取消")
		}
	}
}
