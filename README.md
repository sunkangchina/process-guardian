# windows守护进程 

## 配置

请查看 `config.json`

~~~
{
    "protected_apps": [ 
      {
        "name": "TestPHP",
        "path": "C:\\Users\\test.php",
        "arguments": " ",
        "restart_delay": 10000,
        "max_restarts": 10,
        "auto_start": true
      }
    ],
    "monitor_interval": 3000,
    "log_path": "",
    "enable_file_protection": true
} 
~~~


## 编译为可执行文件

~~~
go build -o process-guardian.exe
~~~

双击 `process-guardian.exe`



# 安装为Windows服务

~~~
@echo off
set SERVICE_NAME=GoProcessGuardian
set EXE_PATH=%~dp0process-guardian.exe

sc create %SERVICE_NAME% binPath= "%EXE_PATH%" start= auto
sc description %SERVICE_NAME% "守护进程服务 - 监控和重启关键应用程序"
sc start %SERVICE_NAME%

echo Service installed and started
pause
~~~
