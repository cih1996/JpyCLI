package cmd

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

// --- 请求/响应结构 ---

type execRequest struct {
	Args []string `json:"args"`
}

type execResponse struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

type shellRequest struct {
	Cmd     string `json:"cmd"`
	Timeout int    `json:"timeout"` // 秒，0=默认120
}

type asyncResponse struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}

type taskResponse struct {
	TaskID   string  `json:"task_id"`
	Status   string  `json:"status"` // running / done / failed
	ExitCode int     `json:"exit_code"`
	Stdout   string  `json:"stdout"`
	Stderr   string  `json:"stderr"`
	Elapsed  string  `json:"elapsed"`
	Command  string  `json:"command"`
	Progress float64 `json:"progress,omitempty"` // 预留
}

// --- 异步任务管理器 ---

type asyncTask struct {
	ID        string
	Cmd       string
	Status    string // running / done / failed
	ExitCode  int
	stdout    bytes.Buffer
	stderr    bytes.Buffer
	StartTime time.Time
	EndTime   time.Time
	cancel    context.CancelFunc
	mu        sync.Mutex
}

func (t *asyncTask) toResponse() taskResponse {
	t.mu.Lock()
	defer t.mu.Unlock()

	elapsed := time.Since(t.StartTime)
	if t.Status != "running" {
		elapsed = t.EndTime.Sub(t.StartTime)
	}

	return taskResponse{
		TaskID:   t.ID,
		Status:   t.Status,
		ExitCode: t.ExitCode,
		Stdout:   t.stdout.String(),
		Stderr:   t.stderr.String(),
		Elapsed:  fmt.Sprintf("%.1fs", elapsed.Seconds()),
		Command:  t.Cmd,
	}
}

type taskManager struct {
	tasks map[string]*asyncTask
	mu    sync.RWMutex
}

func newTaskManager() *taskManager {
	return &taskManager{tasks: make(map[string]*asyncTask)}
}

func (m *taskManager) create(cmdStr string) *asyncTask {
	id := generateID()
	task := &asyncTask{
		ID:        id,
		Cmd:       cmdStr,
		Status:    "running",
		StartTime: time.Now(),
	}
	m.mu.Lock()
	m.tasks[id] = task
	m.mu.Unlock()
	return task
}

func (m *taskManager) get(id string) *asyncTask {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tasks[id]
}

func (m *taskManager) list() []*asyncTask {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*asyncTask, 0, len(m.tasks))
	for _, t := range m.tasks {
		out = append(out, t)
	}
	return out
}

func (m *taskManager) remove(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.tasks[id]; ok {
		delete(m.tasks, id)
		return true
	}
	return false
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// --- Server 命令 ---

func newServerCmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "server",
		Short: "启动 HTTP 代理服务，接收远程 CLI 命令",
		RunE: func(cmd *cobra.Command, args []string) error {
			self, err := os.Executable()
			if err != nil {
				return fmt.Errorf("无法获取自身路径: %v", err)
			}

			tm := newTaskManager()

			mux := http.NewServeMux()
			mux.HandleFunc("/exec", makeExecHandler(self))
			mux.HandleFunc("/exec/async", makeExecAsyncHandler(self, tm))
			mux.HandleFunc("/shell", makeShellHandler(self))
			mux.HandleFunc("/shell/async", makeShellAsyncHandler(self, tm))
			mux.HandleFunc("/shell/task", makeShellTaskHandler(tm))
			mux.HandleFunc("/shell/tasks", makeShellTasksHandler(tm))
			mux.HandleFunc("/shell/kill", makeShellKillHandler(tm))
			mux.HandleFunc("/file/upload", handleFileUpload)
			mux.HandleFunc("/file/download", handleFileDownload)
			mux.HandleFunc("/version", handleVersion)
			mux.HandleFunc("/health", handleHealth)

			addr := fmt.Sprintf(":%d", port)
			fmt.Fprintf(os.Stderr, "jpy server listening on %s\n", addr)

			server := &http.Server{
				Addr:        addr,
				Handler:     mux,
				ReadTimeout: 30 * time.Second,
				// WriteTimeout 不设全局，由各 handler 自行控制
			}
			return server.ListenAndServe()
		},
	}

	cmd.Flags().IntVar(&port, "port", 9090, "监听端口")
	return cmd
}

// --- 同步 /exec（原有，CLI 透传） ---

func makeExecHandler(selfPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

		var req execRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, execResponse{
				ExitCode: 1, Stderr: fmt.Sprintf("invalid request: %v", err),
			})
			return
		}

		if err := validateArgs(req.Args); err != nil {
			writeJSON(w, http.StatusBadRequest, execResponse{
				ExitCode: 1, Stderr: err.Error(),
			})
			return
		}

		var stdout, stderr bytes.Buffer
		cmd := exec.Command(selfPath, req.Args...)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		exitCode := 0
		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = 1
				stderr.WriteString(fmt.Sprintf("\nexec error: %v", err))
			}
		}

		writeJSON(w, http.StatusOK, execResponse{
			ExitCode: exitCode, Stdout: stdout.String(), Stderr: stderr.String(),
		})
	}
}

// --- 异步 /exec/async（异步执行 CLI 命令） ---

func makeExecAsyncHandler(selfPath string, tm *taskManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

		var req struct {
			Args    []string `json:"args"`
			Timeout int      `json:"timeout"` // 秒，0=默认600
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if err := validateArgs(req.Args); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		timeout := 600 * time.Second // 默认 10 分钟
		if req.Timeout > 0 {
			timeout = time.Duration(req.Timeout) * time.Second
		}

		// 构建命令字符串用于显示
		cmdStr := selfPath + " " + strings.Join(req.Args, " ")
		task := tm.create(cmdStr)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		task.cancel = cancel

		go func() {
			defer cancel()

			cmd := exec.CommandContext(ctx, selfPath, req.Args...)
			task.mu.Lock()
			cmd.Stdout = &task.stdout
			cmd.Stderr = &task.stderr
			task.mu.Unlock()

			err := cmd.Run()

			task.mu.Lock()
			defer task.mu.Unlock()
			task.EndTime = time.Now()

			if err != nil {
				if ctx.Err() == context.DeadlineExceeded {
					task.Status = "failed"
					task.ExitCode = 124
					task.stderr.WriteString(fmt.Sprintf("\n命令超时 (%v)", timeout))
				} else if exitErr, ok := err.(*exec.ExitError); ok {
					task.Status = "done"
					task.ExitCode = exitErr.ExitCode()
				} else {
					task.Status = "failed"
					task.ExitCode = 1
					task.stderr.WriteString(fmt.Sprintf("\nexec error: %v", err))
				}
			} else {
				task.Status = "done"
				task.ExitCode = 0
			}
		}()

		writeJSON(w, http.StatusOK, asyncResponse{
			TaskID: task.ID, Status: "running",
		})
	}
}

// --- 同步 /shell（直接执行系统命令） ---

func makeShellHandler(selfPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

		var req shellRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, execResponse{
				ExitCode: 1, Stderr: fmt.Sprintf("invalid request: %v", err),
			})
			return
		}

		if req.Cmd == "" {
			writeJSON(w, http.StatusBadRequest, execResponse{
				ExitCode: 1, Stderr: "cmd 不能为空",
			})
			return
		}

		timeout := 120 * time.Second
		if req.Timeout > 0 {
			timeout = time.Duration(req.Timeout) * time.Second
		}

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		cmd := shellCmd(ctx, req.Cmd)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		exitCode := 0
		if err := cmd.Run(); err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				exitCode = 124 // 与 timeout 命令一致
				stderr.WriteString(fmt.Sprintf("\n命令超时 (%v)", timeout))
			} else if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = 1
				stderr.WriteString(fmt.Sprintf("\nexec error: %v", err))
			}
		}

		writeJSON(w, http.StatusOK, execResponse{
			ExitCode: exitCode, Stdout: stdout.String(), Stderr: stderr.String(),
		})
	}
}

// --- 异步 /shell/async（提交后台任务） ---

func makeShellAsyncHandler(selfPath string, tm *taskManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

		var req shellRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if req.Cmd == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cmd 不能为空"})
			return
		}

		timeout := 600 * time.Second // 异步默认 10 分钟
		if req.Timeout > 0 {
			timeout = time.Duration(req.Timeout) * time.Second
		}

		task := tm.create(req.Cmd)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		task.cancel = cancel

		go func() {
			defer cancel()

			cmd := shellCmd(ctx, req.Cmd)
			task.mu.Lock()
			cmd.Stdout = &task.stdout
			cmd.Stderr = &task.stderr
			task.mu.Unlock()

			err := cmd.Run()

			task.mu.Lock()
			defer task.mu.Unlock()
			task.EndTime = time.Now()

			if err != nil {
				if ctx.Err() == context.DeadlineExceeded {
					task.Status = "failed"
					task.ExitCode = 124
					task.stderr.WriteString(fmt.Sprintf("\n命令超时 (%v)", timeout))
				} else if exitErr, ok := err.(*exec.ExitError); ok {
					task.Status = "done"
					task.ExitCode = exitErr.ExitCode()
				} else {
					task.Status = "failed"
					task.ExitCode = 1
					task.stderr.WriteString(fmt.Sprintf("\nexec error: %v", err))
				}
			} else {
				task.Status = "done"
				task.ExitCode = 0
			}
		}()

		writeJSON(w, http.StatusOK, asyncResponse{
			TaskID: task.ID, Status: "running",
		})
	}
}

// --- /shell/task?id=xxx（查询任务状态+日志） ---

func makeShellTaskHandler(tm *taskManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "缺少 id 参数"})
			return
		}

		task := tm.get(id)
		if task == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "任务不存在"})
			return
		}

		writeJSON(w, http.StatusOK, task.toResponse())
	}
}

// --- /shell/tasks（列出所有任务） ---

func makeShellTasksHandler(tm *taskManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tasks := tm.list()
		out := make([]taskResponse, 0, len(tasks))
		for _, t := range tasks {
			out = append(out, t.toResponse())
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// --- /shell/kill?id=xxx（终止任务） ---

func makeShellKillHandler(tm *taskManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "缺少 id 参数"})
			return
		}

		task := tm.get(id)
		if task == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "任务不存在"})
			return
		}

		task.mu.Lock()
		if task.cancel != nil && task.Status == "running" {
			task.cancel()
			task.Status = "failed"
			task.ExitCode = 137
			task.EndTime = time.Now()
			task.stderr.WriteString("\n任务被手动终止")
		}
		task.mu.Unlock()

		writeJSON(w, http.StatusOK, task.toResponse())
	}
}

// --- 工具函数 ---

func validateArgs(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("args 不能为空")
	}
	for _, arg := range args {
		if arg == "server" {
			return fmt.Errorf("不允许远程执行 server 命令")
		}
		if strings.HasPrefix(arg, "--remote") {
			return fmt.Errorf("不允许远程命令中包含 --remote")
		}
	}
	return nil
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleVersion(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"success": true,
		"data": map[string]string{
			"version":    Version,
			"build_time": BuildTime,
			"git_commit": GitCommit,
			"go_version": runtime.Version(),
			"platform":   runtime.GOOS + "/" + runtime.GOARCH,
		},
	}
	writeJSON(w, http.StatusOK, info)
}

// --- 文件上传 /file/upload ---

type fileResponse struct {
	Success bool   `json:"success"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Error   string `json:"error,omitempty"`
}

func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 禁用读取超时（大文件上传需要更长时间）
	if conn, _, err := w.(http.Hijacker).Hijack(); err == nil {
		// 无法 hijack，使用原始连接
		conn.Close()
	}
	// 通过设置更长的 deadline 来处理大文件
	if ctrl := http.NewResponseController(w); ctrl != nil {
		ctrl.SetReadDeadline(time.Now().Add(30 * time.Minute))
	}

	// 限制上传大小 5GB
	r.Body = http.MaxBytesReader(w, r.Body, 5<<30)

	// 使用流式解析，避免大文件占用内存
	// MaxMemory 设为 32MB，超出部分写临时文件
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, fileResponse{
			Success: false, Error: fmt.Sprintf("解析表单失败: %v", err),
		})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, fileResponse{
			Success: false, Error: fmt.Sprintf("获取文件失败: %v", err),
		})
		return
	}
	defer file.Close()

	destPath := r.FormValue("dest")
	if destPath == "" {
		destPath = header.Filename
	}

	// 如果是相对路径，放到临时目录
	if !filepath.IsAbs(destPath) {
		destPath = filepath.Join(os.TempDir(), destPath)
	}

	// 确保目录存在
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		writeJSON(w, http.StatusInternalServerError, fileResponse{
			Success: false, Error: fmt.Sprintf("创建目录失败: %v", err),
		})
		return
	}

	// 写入文件
	out, err := os.Create(destPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fileResponse{
			Success: false, Error: fmt.Sprintf("创建文件失败: %v", err),
		})
		return
	}
	defer out.Close()

	size, err := io.Copy(out, file)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fileResponse{
			Success: false, Error: fmt.Sprintf("写入文件失败: %v", err),
		})
		return
	}

	writeJSON(w, http.StatusOK, fileResponse{
		Success: true, Path: destPath, Size: size,
	})
}

// --- 文件下载 /file/download ---

type downloadRequest struct {
	URL  string `json:"url"`
	Dest string `json:"dest"`
}

func handleFileDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req downloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, fileResponse{
			Success: false, Error: fmt.Sprintf("解析请求失败: %v", err),
		})
		return
	}

	if req.URL == "" {
		writeJSON(w, http.StatusBadRequest, fileResponse{
			Success: false, Error: "url 不能为空",
		})
		return
	}

	// 解析文件名
	destPath := req.Dest
	if destPath == "" {
		u, err := url.Parse(req.URL)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, fileResponse{
				Success: false, Error: fmt.Sprintf("解析 URL 失败: %v", err),
			})
			return
		}
		destPath = filepath.Base(u.Path)
		if destPath == "" || destPath == "." || destPath == "/" {
			destPath = "downloaded_file"
		}
	}

	// 如果是相对路径，放到临时目录
	if !filepath.IsAbs(destPath) {
		destPath = filepath.Join(os.TempDir(), destPath)
	}

	// 确保目录存在
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		writeJSON(w, http.StatusInternalServerError, fileResponse{
			Success: false, Error: fmt.Sprintf("创建目录失败: %v", err),
		})
		return
	}

	// 下载文件
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(req.URL)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fileResponse{
			Success: false, Error: fmt.Sprintf("下载失败: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		writeJSON(w, http.StatusInternalServerError, fileResponse{
			Success: false, Error: fmt.Sprintf("下载失败: HTTP %d", resp.StatusCode),
		})
		return
	}

	// 写入文件
	out, err := os.Create(destPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fileResponse{
			Success: false, Error: fmt.Sprintf("创建文件失败: %v", err),
		})
		return
	}
	defer out.Close()

	size, err := io.Copy(out, resp.Body)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fileResponse{
			Success: false, Error: fmt.Sprintf("写入文件失败: %v", err),
		})
		return
	}

	writeJSON(w, http.StatusOK, fileResponse{
		Success: true, Path: destPath, Size: size,
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
