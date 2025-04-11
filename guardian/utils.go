package guardian

import (
	"os"
	"path/filepath"
	"fmt"
)

func protectFile(path string) error {
    // 确保文件所在目录存在
    dir := filepath.Dir(path)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return fmt.Errorf("failed to create directory for '%s': %v", path, err)
    }

    // 如果文件不存在，创建一个空文件
    if _, err := os.Stat(path); os.IsNotExist(err) {
        if _, err := os.Create(path); err != nil {
            return fmt.Errorf("failed to create file '%s': %v", path, err)
        }
    } 
	
	// 设置文件为只读
	if err := os.Chmod(path, 0444); err != nil {
		return fmt.Errorf("failed to set file permissions for '%s': %v", path, err)
	}

	return nil
}