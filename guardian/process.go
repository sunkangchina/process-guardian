package guardian

import (
	"fmt"  
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"os"
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
	if app.currentRestarts >= app.MaxRestarts {
		return
	}

	if time.Since(time.Unix(0, app.lastRestart)) < time.Duration(app.RestartDelay)*time.Millisecond {
		return
	}

	var cmd *exec.Cmd
	ext := strings.ToLower(filepath.Ext(app.Path))
	
	switch ext {
	case ".c": // C源码自动编译并执行
		binPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s.exe", app.Name))
		cmd = exec.Command("gcc", app.Path, "-o", binPath)
		if err := cmd.Run(); err != nil {
			return
		}
		cmd = exec.Command(binPath)

	case ".exe": // 已编译的C程序
		args := strings.Fields(app.Arguments)
		cmd = exec.Command(app.Path, args...)

	case ".py":
		cmd = exec.Command("python", app.Path)

	case ".go":
		cmd = exec.Command("go", "run", app.Path)

	case ".php":
		cmd = exec.Command("php", app.Path)

	default: // 其他可执行文件
		cmd = exec.Command(app.Path, strings.Fields(app.Arguments)...)
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}

	if err := cmd.Start(); err != nil {
		return
	}

	app.currentRestarts++
	app.lastRestart = time.Now().UnixNano()

	go func() {
		cmd.Wait()
	}()
}

func (pm *ProcessManager) IsProcessRunning(app *ProtectedApp) bool {
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s", filepath.Base(app.Path)))
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), filepath.Base(app.Path))
}

func (pm *ProcessManager) Monitor() {
	for i := range pm.config.ProtectedApps {
		if !pm.IsProcessRunning(&pm.config.ProtectedApps[i]) {
			pm.StartProcess(&pm.config.ProtectedApps[i])
		}
	}
}

func (pm *ProcessManager) restartProcess(app *ProtectedApp) error {
	if app.currentRestarts >= app.MaxRestarts {
		return fmt.Errorf("maximum restart count reached")
	}

	if time.Since(time.Unix(0, app.lastRestart)) < time.Duration(app.RestartDelay)*time.Millisecond {
		return fmt.Errorf("restart delay not elapsed")
	}

	if app.NeedCompile {
		cmd := exec.Command("go", "build", "-o", app.Path)
		cmd.Dir = filepath.Dir(app.Path)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("compilation failed: %v", err)
		}
	}

	cmd := exec.Command(app.Path, strings.Fields(app.Arguments)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start: %v", err)
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
		return fmt.Errorf("process not running")
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