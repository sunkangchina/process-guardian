package guardian

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type ProcessManager struct {
	config *Config
}

func NewProcessManager(config *Config) *ProcessManager {
	return &ProcessManager{config: config}
}

func (pm *ProcessManager) StartAll() {
	for i := range pm.config.ProtectedApps {
		if pm.config.ProtectedApps[i].AutoStart {
			pm.StartProcess(&pm.config.ProtectedApps[i])
		}
	}
}

func (pm *ProcessManager) StartProcess(app *ProtectedApp) {
	// 检查进程是否已经在运行
	if pm.IsProcessRunning(app) {
		log.Printf("进程 %s 已在运行中，跳过启动", app.Name)
		return
	}

	if app.currentRestarts >= app.MaxRestarts {
		log.Printf("进程 %s 已达到最大重启次数限制", app.Name)
		return
	}

	if time.Since(time.Unix(0, app.lastRestart)) < time.Duration(app.RestartDelay)*time.Millisecond {
		log.Printf("进程 %s 重启延迟时间未到", app.Name)
		return
	}

	var cmd *exec.Cmd
	ext := strings.ToLower(filepath.Ext(app.Path))
	
	switch ext {
	case ".php":
		cmd = exec.Command("php", app.Path)
	case ".py":
		cmd = exec.Command("python", app.Path)
	case ".go":
		cmd = exec.Command("go", "run", app.Path)
	default:
		cmd = exec.Command(app.Path, strings.Fields(app.Arguments)...)
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}

	if err := cmd.Start(); err != nil {
		log.Printf("启动进程 %s 失败：%v", app.Name, err)
		return
	}

	app.Cmd = cmd
	app.currentRestarts++
	app.lastRestart = time.Now().UnixNano()
	log.Printf("已启动进程 %s（进程ID：%d）", app.Name, cmd.Process.Pid)

	go func() {
		cmd.Wait()
		app.Cmd = nil
		log.Printf("进程 %s 已终止运行", app.Name)
	}()
}

func (pm *ProcessManager) IsProcessRunning(app *ProtectedApp) bool {
	// 首先检查我们自己管理的进程
	if app.Cmd != nil && app.Cmd.Process != nil {
		// 使用 tasklist 检查 PID 是否存在
		cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", app.Cmd.Process.Pid), "/FO", "CSV", "/NH")
		output, err := cmd.Output()
		if err == nil && strings.Contains(string(output), fmt.Sprintf("%d", app.Cmd.Process.Pid)) {
			return true
		}
	}

	// 如果我们的进程不存在，检查是否有同名进程
	processName := filepath.Base(app.Path)
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s", processName), "/FO", "CSV", "/NH")
	output, err := cmd.Output()
	if err == nil && strings.Contains(string(output), processName) {
		return true
	}

	return false
}

func (pm *ProcessManager) Monitor() {
	for i := range pm.config.ProtectedApps {
		app := &pm.config.ProtectedApps[i]
		
		// 如果进程已经在运行，继续监控下一个
		if pm.IsProcessRunning(app) {
			continue
		}
		
		// 如果进程不在运行，且允许自动启动，则尝试启动
		if app.AutoStart {
			// 检查重启限制
			if app.currentRestarts >= app.MaxRestarts {
				continue
			}
			
			// 检查重启延迟
			if time.Since(time.Unix(0, app.lastRestart)) < time.Duration(app.RestartDelay)*time.Millisecond {
				continue
			}
			
			log.Printf("检测到进程 %s 未运行，准备启动", app.Name)
			pm.StartProcess(app)
		}
	}
}

func (pm *ProcessManager) restartProcess(app *ProtectedApp) error {
	if app.currentRestarts >= app.MaxRestarts {
		return fmt.Errorf("进程已达到最大重启次数")
	}

	if time.Since(time.Unix(0, app.lastRestart)) < time.Duration(app.RestartDelay)*time.Millisecond {
		return fmt.Errorf("重启延迟时间未到")
	}

	if app.NeedCompile {
		cmd := exec.Command("go", "build", "-o", app.Path)
		cmd.Dir = filepath.Dir(app.Path)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("编译失败: %v", err)
		}
	}

	cmd := exec.Command(app.Path, strings.Fields(app.Arguments)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动失败: %v", err)
	}

	app.currentRestarts++
	app.lastRestart = time.Now().UnixNano()

	go func() {
		cmd.Wait()
	}()

	return nil
}

func (pm *ProcessManager) checkProcess(app *ProtectedApp) error {
	if app.Cmd == nil || app.Cmd.Process == nil {
		return fmt.Errorf("进程未运行")
	}

	process, err := os.FindProcess(app.Cmd.Process.Pid)
	if err != nil {
		return err
	}

	err = process.Signal(syscall.Signal(0))
	if err != nil {
		return err
	}

	return nil
}

func (pm *ProcessManager) StopAll() {
	for i := range pm.config.ProtectedApps {
		pm.StopProcess(&pm.config.ProtectedApps[i])
	}
}

func (pm *ProcessManager) StopProcess(app *ProtectedApp) {
	if app.Cmd != nil && app.Cmd.Process != nil {
		log.Printf("正在停止进程 %s（进程ID：%d）", app.Name, app.Cmd.Process.Pid)
		
		// 先尝试优雅关闭
		_ = app.Cmd.Process.Signal(syscall.SIGTERM)
		
		// 等待一小段时间
		time.Sleep(time.Second)
		
		// 如果进程还在运行，强制终止
		if pm.IsProcessRunning(app) {
			processName := filepath.Base(app.Path)
			cmd := exec.Command("taskkill", "/F", "/IM", processName)
			_ = cmd.Run()
		}
	}
	
	app.Cmd = nil
	log.Printf("进程 %s 已停止", app.Name)
}