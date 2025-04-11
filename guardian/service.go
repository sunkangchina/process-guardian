package guardian

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

type Daemon struct {
	config         *Config
	processManager *ProcessManager
	stopChan       chan struct{}
	logFile        *os.File
}

func NewDaemon(configPath string) (*Daemon, error) {
    config, err := LoadConfig(configPath)
    if err != nil {
        return nil, fmt.Errorf("failed to load config: %v", err)
    }

    // 如果日志路径为空，使用默认路径
    if config.LogPath == "" {
        config.LogPath = "logs/process-guardian.log"
    }

    // 确保日志目录存在
    logDir := filepath.Dir(config.LogPath)
    if err := os.MkdirAll(logDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create log directory '%s': %v", logDir, err)
    }

    // 尝试创建/打开日志文件
    logFile, err := os.OpenFile(config.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return nil, fmt.Errorf("failed to open log file '%s': %v", config.LogPath, err)
    }

    // 设置日志输出
    log.SetOutput(logFile)
    log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
    log.Printf("Process Guardian started (log path: %s)", config.LogPath)

    return &Daemon{
        config:         config,
        processManager: NewProcessManager(config),
        stopChan:       make(chan struct{}),
        logFile:        logFile,
    }, nil
}

func (d *Daemon) initLogger(config *Config) error {
	if config.LogPath == "" {
		return nil
	}
	logFile, err := os.OpenFile(config.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	d.logFile = logFile
	return nil
}

func (d *Daemon) Start() error {
	log.Println("Starting process guardian")

	if d.config.EnableFileProtection {
		d.setupFileProtection()
	}

	d.processManager.StartAll()
	go d.monitorLoop()

	return d.protectAll()
}

func (d *Daemon) Stop() {
	close(d.stopChan)
	log.Println("Stopping process guardian")
	
	// 停止所有进程
	d.processManager.StopAll()
	
	// 关闭日志文件
	if d.logFile != nil {
		d.logFile.Close()
	}
}

func (d *Daemon) Pause() {
	log.Println("Pausing process guardian")
	d.processManager.StopAll()
}

func (d *Daemon) Resume() error {
	log.Println("Resuming process guardian")
	return d.Start()
}

func (d *Daemon) monitorLoop() {
	ticker := time.NewTicker(time.Duration(d.config.MonitorInterval) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.processManager.Monitor()
		case <-d.stopChan:
			return
		}
	}
}

func (d *Daemon) setupFileProtection() {
	for _, app := range d.config.ProtectedApps {
		if err := protectFile(app.Path); err != nil {
			log.Printf("Failed to protect %s: %v", app.Path, err)
		}
	}
	
	if err := protectFile("config.json"); err != nil {
		log.Printf("Failed to protect config file: %v", err)
	}
}

func (d *Daemon) protectApp(app *ProtectedApp) error {
	if d.config.EnableFileProtection {
		return protectFile(app.Path)
	}
	return nil
}

func (d *Daemon) protectConfig() error {
	if d.config.EnableFileProtection {
		return protectFile("config.json")
	}
	return nil
}

func (d *Daemon) protectAll() error {
	for _, app := range d.config.ProtectedApps {
		if err := d.protectApp(&app); err != nil {
			continue
		}
	}

	if err := d.protectConfig(); err != nil {
		return err
	}

	return nil
}