# windows守护进程 

# src 目录

## 配置

1. 请查看 `config.json`

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


2. 编译为可执行文件

~~~
go build -ldflags "-H windowsgui -s -w" -o process-guardian.exe
~~~

双击 `process-guardian.exe`


将在托盘区看到守护进程的图标，点击图标有退出按钮

## 复制生成的exe文件至bin目录

`src`下的

- icon.icon  
- config.json

需要一起复制过去。


## 演示效果

![](./doc/1.jpg)

![](./doc/2.jpg)


php 代码演示

~~~
<?php 
while (true) {
   file_put_contents(__DIR__.'/1.txt', date("Y-m-d H:i:s")."\n",FILE_APPEND);
   sleep(1);
}
~~~