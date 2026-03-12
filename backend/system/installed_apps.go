package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type InstalledApp struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Version string `json:"version,omitempty"`
	Vendor  string `json:"vendor,omitempty"`
}

func (s *SystemExecutor) ScanInstalledApps() ([]InstalledApp, error) {
	if !s.perm.CanExecute("scan_apps") {
		return nil, fmt.Errorf("permission denied")
	}

	var apps []InstalledApp

	registryPaths := []string{
		`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`HKLM\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
		`HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
	}

	for _, regPath := range registryPaths {
		apps = s.scanRegistryPath(regPath, apps)
	}

	apps = s.deduplicateApps(apps)

	return apps, nil
}

func (s *SystemExecutor) scanRegistryPath(regPath string, apps []InstalledApp) []InstalledApp {
	cmd := exec.Command("reg", "query", regPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return apps
	}

	escapedPath := strings.ReplaceAll(regPath, `\`, `\\`)
	subkeyRegex := regexp.MustCompile(`^\s+` + escapedPath + `\\(\S+)\s*$`)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		match := subkeyRegex.FindStringSubmatch(strings.TrimSpace(line))
		if len(match) > 1 {
			subkey := match[1]
			fullPath := fmt.Sprintf(`%s\%s`, regPath, subkey)
			app := s.getAppDetails(fullPath)
			if app.Name != "" && app.Path != "" {
				apps = append(apps, app)
			}
		}
	}

	return apps
}

func (s *SystemExecutor) getAppDetails(regPath string) InstalledApp {
	app := InstalledApp{}

	output, _ := exec.Command("cmd", "/c", fmt.Sprintf(`reg query "%s" /v DisplayName 2>nul`, regPath)).CombinedOutput()
	outputStr := string(output)

	nameMatch := regexp.MustCompile(`DisplayName\s+REG_SZ\s+(.+)`).FindStringSubmatch(outputStr)
	if len(nameMatch) > 1 {
		app.Name = strings.TrimSpace(nameMatch[1])
	} else {
		return app
	}

	_ = exec.Command("reg", "query", regPath, "/v", "InstallLocation")
	output, _ = exec.Command("cmd", "/c", fmt.Sprintf(`reg query "%s" /v InstallLocation 2>nul`, regPath)).CombinedOutput()
	pathMatch := regexp.MustCompile(`InstallLocation\s+REG_SZ\s+(.+)`).FindStringSubmatch(string(output))
	if len(pathMatch) > 1 {
		app.Path = strings.TrimSpace(pathMatch[1])
	}

	_ = exec.Command("reg", "query", regPath, "/v", "DisplayVersion")
	output, _ = exec.Command("cmd", "/c", fmt.Sprintf(`reg query "%s" /v DisplayVersion 2>nul`, regPath)).CombinedOutput()
	versionMatch := regexp.MustCompile(`DisplayVersion\s+REG_SZ\s+(.+)`).FindStringSubmatch(string(output))
	if len(versionMatch) > 1 {
		app.Version = strings.TrimSpace(versionMatch[1])
	}

	_ = exec.Command("reg", "query", regPath, "/v", "Publisher")
	output, _ = exec.Command("cmd", "/c", fmt.Sprintf(`reg query "%s" /v Publisher 2>nul`, regPath)).CombinedOutput()
	vendorMatch := regexp.MustCompile(`Publisher\s+REG_SZ\s+(.+)`).FindStringSubmatch(string(output))
	if len(vendorMatch) > 1 {
		app.Vendor = strings.TrimSpace(vendorMatch[1])
	}

	return app
}

func (s *SystemExecutor) deduplicateApps(apps []InstalledApp) []InstalledApp {
	seen := make(map[string]bool)
	var result []InstalledApp

	for _, app := range apps {
		key := strings.ToLower(app.Name)
		if !seen[key] {
			seen[key] = true
			result = append(result, app)
		}
	}

	return result
}

func (s *SystemExecutor) LaunchApp(appPath string) error {
	if !s.perm.CanExecute("run_command") {
		return fmt.Errorf("permission denied")
	}

	if _, err := os.Stat(appPath); err != nil {
		return fmt.Errorf("app not found: %s", appPath)
	}

	cmd := exec.Command("cmd", "/c", "start", "", appPath)
	return cmd.Start()
}

func (s *SystemExecutor) FindExecutable(appName string) (string, error) {
	commonPaths := s.getCommonAppPaths(appName)
	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	apps, err := s.ScanInstalledApps()
	if err == nil && len(apps) > 0 {
		searchName := strings.ToLower(appName)
		for _, app := range apps {
			if strings.Contains(strings.ToLower(app.Name), searchName) {
				exePath, err := s.findMainExecutable(app.Path)
				if err == nil && exePath != "" {
					return exePath, nil
				}
			}
		}
	}

	return "", fmt.Errorf("executable not found for: %s", appName)
}

func (s *SystemExecutor) getCommonAppPaths(appName string) []string {
	searchName := strings.ToLower(appName)
	var paths []string

	if strings.Contains(searchName, "word") || strings.Contains(searchName, "office") {
		paths = append(paths,
			`C:\Program Files\Microsoft Office\root\Office16\WINWORD.EXE`,
			`C:\Program Files\Microsoft Office\root\Office15\WINWORD.EXE`,
			`C:\Program Files\Microsoft Office 16\root\Office16\WINWORD.EXE`,
			`C:\Program Files (x86)\Microsoft Office\root\Office16\WINWORD.EXE`,
			`C:\Program Files (x86)\Microsoft Office\root\Office15\WINWORD.EXE`,
			`C:\Program Files (x86)\Microsoft Office 16\root\Office16\WINWORD.EXE`,
			`C:\Program Files\Microsoft Office 365\root\Office16\WINWORD.EXE`,
		)
	}

	if strings.Contains(searchName, "excel") {
		paths = append(paths,
			`C:\Program Files\Microsoft Office\root\Office16\EXCEL.EXE`,
			`C:\Program Files\Microsoft Office\root\Office15\EXCEL.EXE`,
			`C:\Program Files (x86)\Microsoft Office\root\Office16\EXCEL.EXE`,
		)
	}

	if strings.Contains(searchName, "powerpoint") || strings.Contains(searchName, "ppt") {
		paths = append(paths,
			`C:\Program Files\Microsoft Office\root\Office16\POWERPNT.EXE`,
			`C:\Program Files\Microsoft Office\root\Office15\POWERPNT.EXE`,
		)
	}

	if strings.Contains(searchName, "notepad") {
		paths = append(paths, `C:\Windows\System32\notepad.exe`)
	}

	return paths
}

func (s *SystemExecutor) findMainExecutable(dir string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("empty directory")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	exeFiles := []string{}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".exe") {
			exeFiles = append(exeFiles, e.Name())
		}
	}

	if len(exeFiles) > 0 {
		return filepath.Join(dir, exeFiles[0]), nil
	}

	for _, e := range entries {
		if e.IsDir() {
			subPath := filepath.Join(dir, e.Name())
			if result, err := s.findMainExecutable(subPath); err == nil {
				return result, nil
			}
		}
	}

	return "", fmt.Errorf("no executable found")
}
