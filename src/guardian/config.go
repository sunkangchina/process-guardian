package guardian

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

type ProtectedApp struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	Arguments     string `json:"arguments"`
	RestartDelay  int    `json:"restart_delay"`  // 毫秒
	MaxRestarts   int    `json:"max_restarts"`
	AutoStart     bool   `json:"auto_start"`
	NeedCompile   bool   `json:"need_compile"`
	currentRestarts int
	lastRestart   int64
	Cmd           *exec.Cmd
}

type Config struct {
	ProtectedApps       []ProtectedApp `json:"protected_apps"`
	MonitorInterval    int           `json:"monitor_interval"` // 毫秒
	LogPath           string        `json:"log_path"`
	EnableFileProtection bool        `json:"enable_file_protection"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(file, &config); err != nil {
		return nil, err
	}

	// 确保日志目录存在
	logDir := filepath.Dir(config.LogPath)
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		os.MkdirAll(logDir, 0755)
	}

	return &config, nil
}