package utils

import (
	"fmt"
	"os"
)

func ReadFile(path string) ([]byte, error) {
      data, err := os.ReadFile(path)
      if err != nil {
          if os.IsNotExist(err) {
              return nil, &NotFoundError{Message: fmt.Sprintf("%s not found", path)}
          }
          return nil, &InternalError{Message: fmt.Sprintf("read %s: %s", path, err.Error())}
      }
      return data, nil
  }

func WriteFile(path string, data []byte, mode os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
