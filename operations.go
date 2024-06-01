package main

import (
  "context"
  "fmt"
  "os"
  "os/exec"
  "path/filepath"
  "strings"

  "gollama/logging"

  "github.com/ollama/ollama/api"
  "golang.org/x/term"
)

func runModel(modelName string) {
  // Save the current terminal state
  oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
  if err != nil {
    logging.ErrorLogger.Printf("Error saving terminal state: %v\n", err)
    return
  }
  defer term.Restore(int(os.Stdin.Fd()), oldState)

  // Clear the terminal screen
  fmt.Print("\033[H\033[2J")

  // Run the Ollama model
  cmd := exec.Command("ollama", "run", modelName)
  cmd.Stdout = os.Stdout
  cmd.Stderr = os.Stderr
  cmd.Stdin = os.Stdin

  if err := cmd.Run(); err != nil {
    logging.ErrorLogger.Printf("Error running model: %v\n", err)
  } else {
    logging.InfoLogger.Printf("Successfully ran model: %s\n", modelName)
  }

  // Restore the terminal state
  if err := term.Restore(int(os.Stdin.Fd()), oldState); err != nil {
    logging.ErrorLogger.Printf("Error restoring terminal state: %v\n", err)
  }

  // Clear the terminal screen again to refresh the application view
  fmt.Print("\033[H\033[2J")
}

func deleteModel(client *api.Client, name string) error {
  ctx := context.Background()
  req := &api.DeleteRequest{Name: name}
  logging.DebugLogger.Printf("Attempting to delete model: %s\n", name)

  // Log the request details
  logging.DebugLogger.Printf("Delete request: %+v\n", req)

  err := client.Delete(ctx, req)
  if err != nil {
    // Print a detailed error message to the console
    logging.ErrorLogger.Printf("Error deleting model %s: %v\n", name, err)
    // Return an error so that it can be handled by the calling function
    return fmt.Errorf("error deleting model %s: %v", name, err)
  }

  // If we reach this point, the model was deleted successfully
  logging.InfoLogger.Printf("Successfully deleted model: %s\n", name)
  return nil
}

func linkModel(modelName, lmStudioModelsDir string, noCleanup bool) (string, error) {
  modelPath, err := getModelPath(modelName)
  if err != nil {
    return "", fmt.Errorf("error getting model path for %s: %v", modelName, err)
  }

  parts := strings.Split(modelName, ":")
  author := "unknown"
  if len(parts) > 1 {
    author = strings.ReplaceAll(parts[0], "/", "-")
  }

  lmStudioModelName := strings.ReplaceAll(strings.ReplaceAll(modelName, ":", "-"), "_", "-")
  lmStudioModelDir := filepath.Join(lmStudioModelsDir, author, lmStudioModelName+"-GGUF")

  // Check if the model path is a valid file
  fileInfo, err := os.Stat(modelPath)
  if err != nil || fileInfo.IsDir() {
    return "", fmt.Errorf("invalid model path for %s: %s", modelName, modelPath)
  }

  // Check if the symlink already exists and is valid
  lmStudioModelPath := filepath.Join(lmStudioModelDir, filepath.Base(lmStudioModelName)+".gguf")
  if _, err := os.Lstat(lmStudioModelPath); err == nil {
    if isValidSymlink(lmStudioModelPath, modelPath) {
      message := "Model %s is already symlinked to %s"
      logging.InfoLogger.Printf(message+"\n", modelName, lmStudioModelPath)
      return "", nil
    }
    // Remove the invalid symlink
    err = os.Remove(lmStudioModelPath)
    if err != nil {
      message := "failed to remove invalid symlink %s: %v"
      logging.ErrorLogger.Printf(message+"\n", lmStudioModelPath, err)
      return "", fmt.Errorf(message, lmStudioModelPath, err)
    }
  }

  // Check if the model is already symlinked in another location
  var existingSymlinkPath string
  err = filepath.Walk(lmStudioModelsDir, func(path string, info os.FileInfo, err error) error {
    if err != nil {
      return err
    }
    if info.Mode()&os.ModeSymlink != 0 {
      linkPath, err := os.Readlink(path)
      if err != nil {
        return err
      }
      if linkPath == modelPath {
        existingSymlinkPath = path
        return nil
      }
    }
    return nil
  })
  if err != nil {
    message := "error walking LM Studio models directory: %v"
    logging.ErrorLogger.Printf(message+"\n", err)
    return "", fmt.Errorf(message, err)
  }

  if existingSymlinkPath != "" {
    // Remove the duplicated model directory
    err = os.RemoveAll(lmStudioModelDir)
    if err != nil {
      message := "failed to remove duplicated model directory %s: %v"
      logging.ErrorLogger.Printf(message+"\n", lmStudioModelDir, err)
      return "", fmt.Errorf(message, lmStudioModelDir, err)
    }
    return fmt.Sprintf("Removed duplicated model directory %s", lmStudioModelDir), nil
  }

  // Create the symlink
  err = os.MkdirAll(lmStudioModelDir, os.ModePerm)
  if err != nil {
    message := "failed to create directory %s: %v"
    logging.ErrorLogger.Printf(message+"\n", lmStudioModelDir, err)
    return "", fmt.Errorf(message, lmStudioModelDir, err)
  }
  err = os.Symlink(modelPath, lmStudioModelPath)
  if err != nil {
    message := "failed to symlink %s: %v"
    logging.ErrorLogger.Printf(message+"\n", modelName, err)
    return "", fmt.Errorf(message, modelName, err)
  }
  if !noCleanup {
    cleanBrokenSymlinks(lmStudioModelsDir)
  }
  message := "Symlinked %s to %s"
  logging.InfoLogger.Printf(message+"\n", modelName, lmStudioModelPath)
  return "", nil
}

func getModelPath(modelName string) (string, error) {
  cmd := exec.Command("ollama", "show", "--modelfile", modelName)
  output, err := cmd.Output()
  if err != nil {
    return "", err
  }
  lines := strings.Split(strings.TrimSpace(string(output)), "\n")
  for _, line := range lines {
    if strings.HasPrefix(line, "FROM ") {
      return strings.TrimSpace(line[5:]), nil
    }
  }
  message := "failed to get model path for %s: no 'FROM' line in output"
  logging.ErrorLogger.Printf(message+"\n", modelName)
  return "", fmt.Errorf(message, modelName)
}

func cleanBrokenSymlinks(lmStudioModelsDir string) {
  err := filepath.Walk(lmStudioModelsDir, func(path string, info os.FileInfo, err error) error {
    if err != nil {
      return err
    }
    if info.IsDir() {
      files, err := os.ReadDir(path)
      if err != nil {
        return err
      }
      if len(files) == 0 {
        logging.InfoLogger.Printf("Removing empty directory: %s\n", path)
        err = os.Remove(path)
        if err != nil {
          return err
        }
      }
    } else if info.Mode()&os.ModeSymlink != 0 {
      linkPath, err := os.Readlink(path)
      if err != nil {
        return err
      }
      if !isValidSymlink(path, linkPath) {
        logging.InfoLogger.Printf("Removing invalid symlink: %s\n", path)
        err = os.Remove(path)
        if err != nil {
          return err
        }
      }
    }
    return nil
  })
  if err != nil {
    logging.ErrorLogger.Printf("Error walking LM Studio models directory: %v\n", err)
    return
  }
}

func isValidSymlink(symlinkPath, targetPath string) bool {
  // Check if the symlink matches the expected naming convention
  expectedSuffix := ".gguf"
  if !strings.HasSuffix(filepath.Base(symlinkPath), expectedSuffix) {
    return false
  }

  // Check if the target file exists
  if _, err := os.Stat(targetPath); os.IsNotExist(err) {
    return false
  }

  // Check if the symlink target is a file (not a directory or another symlink)
  fileInfo, err := os.Lstat(targetPath)
  if err != nil || fileInfo.Mode()&os.ModeSymlink != 0 || fileInfo.IsDir() {
    return false
  }

  return true
}

func cleanupSymlinkedModels(lmStudioModelsDir string) {
  for {
    hasEmptyDir := false
    err := filepath.Walk(lmStudioModelsDir, func(path string, info os.FileInfo, err error) error {
      if err != nil {
        return err
      }
      if info.IsDir() {
        files, err := os.ReadDir(path)
        if err != nil {
          return err
        }
        if len(files) == 0 {
          logging.InfoLogger.Printf("Removing empty directory: %s\n", path)
          err = os.Remove(path)
          if err != nil {
            return err
          }
          hasEmptyDir = true
        }
      } else if info.Mode()&os.ModeSymlink != 0 {
        logging.InfoLogger.Printf("Removing symlinked model: %s\n", path)
        err = os.Remove(path)
        if err != nil {
          return err
        }
      }
      return nil
    })
    if err != nil {
      logging.ErrorLogger.Printf("Error walking LM Studio models directory: %v\n", err)
      return
    }
    if !hasEmptyDir {
      break
    }
  }
}
