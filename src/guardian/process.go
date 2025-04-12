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

const (
	CREATE_NO_WINDOW         = 0x08000000
	DETACHED_PROCESS        = 0x00000008
	CREATE_BREAKAWAY_FROM_JOB = 0x01000000
)

type ProcessManager struct {
	config *Config
}

// setProcessAttributes 设置进程的启动属性，确保不显示窗口
func setProcessAttributes(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: CREATE_NO_WINDOW | DETACHED_PROCESS | CREATE_BREAKAWAY_FROM_JOB,
	}
	// 确保不继承标准输入输出
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
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

	// 设置进程属性，确保不显示窗口
	setProcessAttributes(cmd)

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
	if app.Cmd != nil && app.Cmd.Process != nil {
		cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", app.Cmd.Process.Pid), "/FO", "CSV", "/NH")
		setProcessAttributes(cmd)
		output, err := cmd.Output()
		if err == nil && strings.Contains(string(output), fmt.Sprintf("%d", app.Cmd.Process.Pid)) {
			return true
		}
	}

	processName := filepath.Base(app.Path)
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s", processName), "/FO", "CSV", "/NH")
	setProcessAttributes(cmd)
	output, err := cmd.Output()
	if err == nil && strings.Contains(string(output), processName) {
		return true
	}

	return false
}

func (pm *ProcessManager) Monitor() {
	for i := range pm.config.ProtectedApps {
		app := &pm.config.ProtectedApps[i]
		
		if pm.IsProcessRunning(app) {
			continue
		}
		
		if app.AutoStart {
			if app.currentRestarts >= app.MaxRestarts {
				continue
			}
			
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
		compileCmd := exec.Command("go", "build", "-o", app.Path)
		compileCmd.Dir = filepath.Dir(app.Path)
		setProcessAttributes(compileCmd)
		if err := compileCmd.Run(); err != nil {
			return fmt.Errorf("编译失败: %v", err)
		}
	}

	cmd := exec.Command(app.Path, strings.Fields(app.Arguments)...)
	setProcessAttributes(cmd)

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
			killCmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", app.Cmd.Process.Pid))
			setProcessAttributes(killCmd)
			_ = killCmd.Run()
		}
	}
	
	app.Cmd = nil
	log.Printf("进程 %s 已停止", app.Name)
}