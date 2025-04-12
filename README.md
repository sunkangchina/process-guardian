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
go build -ldflags "-H windowsgui -s -w" -o process-guardian.exe
~~~

双击 `process-guardian.exe`


 