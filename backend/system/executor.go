package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type SystemExecutor struct {
	perm *PermissionManager
}

func NewSystemExecutor(perm *PermissionManager) *SystemExecutor {
	return &SystemExecutor{perm: perm}
}

func (s *SystemExecutor) RunCommand(cmd string) (string, error) {
	if !s.perm.CanExecute("run_command") {
		return "", fmt.Errorf("permission denied")
	}

	fmt.Printf("[DEBUG] RunCommand: %s\n", cmd)

	execmd := exec.Command("cmd", "/c", "start", "", cmd)
	output, err := execmd.CombinedOutput()
	if err != nil {
		fmt.Printf("[DEBUG] RunCommand error: %v, output: %s\n", err, string(output))
	}
	return string(output), err
}

func (s *SystemExecutor) OpenBrowser(url string) error {
	if !s.perm.CanExecute("open_browser") {
		return fmt.Errorf("permission denied")
	}
	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}

	cmd := exec.Command("cmd", "/c", "start", "", url)
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to open browser: %v", err)
	}
	return nil
}

func (s *SystemExecutor) ReadFile(path string) (string, error) {
	if !s.perm.CanExecute("read_file") {
		return "", fmt.Errorf("permission denied")
	}
	data, err := os.ReadFile(path)
	return string(data), err
}

func (s *SystemExecutor) WriteFile(path, content string) error {
	if !s.perm.CanExecute("write_file") {
		return fmt.Errorf("permission denied")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func (s *SystemExecutor) ListDir(path string) ([]string, error) {
	if !s.perm.CanExecute("list_dir") {
		return nil, fmt.Errorf("permission denied")
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var result []string
	for _, e := range entries {
		result = append(result, e.Name())
	}
	return result, nil
}

func (s *SystemExecutor) GetSystemInfo() (string, error) {
	if !s.perm.CanExecute("system_info") {
		return "", fmt.Errorf("permission denied")
	}
	info := fmt.Sprintf("OS: %s\n", runtime.GOOS)
	info += fmt.Sprintf("Arch: %s\n", runtime.GOARCH)
	return info, nil
}
