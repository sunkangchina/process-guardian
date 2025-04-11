package guardian

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

const maxLogSize = 10 * 1024 * 1024 // 10MB

func protectFile(path string) error {
    // 确保文件所在目录存在
    dir := filepath.Dir(path)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return fmt.Errorf("创建目录 '%s' 失败: %v", dir, err)
    }

    // 如果文件不存在，创建一个空文件
    if _, err := os.Stat(path); os.IsNotExist(err) {
        if _, err := os.Create(path); err != nil {
            return fmt.Errorf("创建文件 '%s' 失败: %v", path, err)
        }
    } 
	
	// 设置文件为只读
	if err := os.Chmod(path, 0444); err != nil {
		return fmt.Errorf("设置文件 '%s' 权限失败: %v", path, err)
	}

	return nil
}

func checkAndRotateLog(logPath string) error {
    // 检查文件是否存在
    info, err := os.Stat(logPath)
    if err != nil {
        if os.IsNotExist(err) {
            return nil
        }
        return err
    }

    // 如果文件大小超过10MB，清空文件
    if info.Size() > maxLogSize {
        // 直接清空文件而不是删除，这样可以保持文件句柄有效
        if err := os.Truncate(logPath, 0); err != nil {
            return fmt.Errorf("清空日志文件失败: %v", err)
        }
        log.Printf("日志文件已超过10MB，已清空旧日志")
    }
    return nil
}