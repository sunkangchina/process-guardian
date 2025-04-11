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
	isPaused       bool
}

func NewDaemon(configPath string) (*Daemon, error) {
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("加载配置文件失败: %v", err)
	}

	// 如果日志路径为空，使用默认路径
	if config.LogPath == "" {
		config.LogPath = "logs/process-guardian.log"
	}

	// 确保日志目录存在
	logDir := filepath.Dir(config.LogPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录 '%s' 失败: %v", logDir, err)
	}

	// 尝试创建/打开日志文件
	logFile, err := os.OpenFile(config.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件 '%s' 失败: %v", config.LogPath, err)
	}

	// 设置日志输出
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Printf("进程守护程序已启动（日志路径：%s）", config.LogPath)

	return &Daemon{
		config:         config,
		processManager: NewProcessManager(config),
		stopChan:       make(chan struct{}),
		logFile:        logFile,
		isPaused:       false,
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
	log.Println("正在启动进程守护程序")

	if d.config.EnableFileProtection {
		d.setupFileProtection()
	}

	d.processManager.StartAll()
	go d.monitorLoop()

	return d.protectAll()
}

func (d *Daemon) Stop() {
	log.Println("正在停止进程守护程序")
	// 先关闭监控循环
	close(d.stopChan)
	// 等待一下确保监控循环已经退出
	time.Sleep(time.Second)
	
	// 然后停止所有进程
	d.processManager.StopAll()
	
	// 最后关闭日志文件
	if d.logFile != nil {
		d.logFile.Close()
	}
}

func (d *Daemon) Pause() {
	log.Println("正在暂停进程守护程序")
	d.isPaused = true
	d.processManager.StopAll()
}

func (d *Daemon) Resume() error {
	log.Println("正在恢复进程守护程序")
	d.isPaused = false
	d.processManager.StartAll()
	return nil
}

func (d *Daemon) monitorLoop() {
	ticker := time.NewTicker(time.Duration(d.config.MonitorInterval) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-d.stopChan:
			// 收到停止信号立即退出
			log.Println("正在退出监控循环")
			return
		case <-ticker.C:
			if !d.isPaused {
				d.processManager.Monitor()
			}
		}
	}
}

func (d *Daemon) setupFileProtection() {
	for _, app := range d.config.ProtectedApps {
		if err := protectFile(app.Path); err != nil {
			log.Printf("保护文件 %s 失败: %v", app.Path, err)
		}
	}
	
	if err := protectFile("config.json"); err != nil {
		log.Printf("保护配置文件失败: %v", err)
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