package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type Skill struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Params      []SkillParam `json:"params"`
	IsPython    bool         `json:"is_python"`
	Metadata    string       `json:"metadata,omitempty"`
}

type SkillParam struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

type SkillManager struct {
	skills    map[string]SkillFunc
	pythonDir string
	skillMeta map[string]Skill
}

type SkillFunc func(ctx context.Context, params map[string]interface{}) (string, error)

func NewSkillManager(pythonDir string) *SkillManager {
	m := &SkillManager{
		skills:    make(map[string]SkillFunc),
		pythonDir: pythonDir,
		skillMeta: make(map[string]Skill),
	}
	m.registerBuiltins()
	m.scanPythonSkills()
	return m
}

func (m *SkillManager) scanPythonSkills() {
	if m.pythonDir == "" {
		return
	}

	entries, err := os.ReadDir(m.pythonDir)
	if err != nil {
		return
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		scriptPath := filepath.Join(m.pythonDir, name, name+".py")
		if _, err := os.Stat(scriptPath); err == nil {
			m.RegisterPythonSkill(name)
			m.loadSkillMetadata(name)
		}
	}
}

func (m *SkillManager) loadSkillMetadata(name string) {
	skillPath := filepath.Join(m.pythonDir, name, "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		return
	}

	content := string(data)
	re := regexp.MustCompile(`---\s*\n([\s\S]*?)\n---`)
	matches := re.FindStringSubmatch(content)
	if len(matches) < 2 {
		return
	}

	frontmatter := matches[1]

	nameRe := regexp.MustCompile(`(?m)^name:\s*(.+)$`)
	nameMatch := nameRe.FindStringSubmatch(frontmatter)
	skillName := name
	if len(nameMatch) > 1 {
		skillName = strings.TrimSpace(nameMatch[1])
	}

	descRe := regexp.MustCompile(`(?m)^description:\s*(.+)$`)
	descMatch := descRe.FindStringSubmatch(frontmatter)
	skillDesc := fmt.Sprintf("Python skill: %s", name)
	if len(descMatch) > 1 {
		skillDesc = strings.TrimSpace(descMatch[1])
	}

	m.skillMeta[name] = Skill{
		Name:        skillName,
		Description: skillDesc,
		IsPython:    true,
	}
}

func (m *SkillManager) registerBuiltins() {
	m.skills["file_read"] = func(ctx context.Context, params map[string]interface{}) (string, error) {
		path, _ := params["path"].(string)
		if path == "" {
			return "", fmt.Errorf("path is required")
		}
		data, err := os.ReadFile(path)
		return string(data), err
	}

	m.skills["file_write"] = func(ctx context.Context, params map[string]interface{}) (string, error) {
		path, _ := params["path"].(string)
		content, _ := params["content"].(string)
		if path == "" {
			return "", fmt.Errorf("path is required")
		}
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory: %v", err)
		}
		err := os.WriteFile(path, []byte(content), 0644)
		return "File written successfully", err
	}

	m.skills["file_list"] = func(ctx context.Context, params map[string]interface{}) (string, error) {
		path, _ := params["path"].(string)
		if path == "" {
			path = "."
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return "", err
		}
		var result []string
		for _, e := range entries {
			result = append(result, e.Name())
		}
		return strings.Join(result, "\n"), nil
	}

	m.skills["system_info"] = func(ctx context.Context, params map[string]interface{}) (string, error) {
		return "WinClaw AI Agent - System Information\n", nil
	}

	m.skills["calculator"] = func(ctx context.Context, params map[string]interface{}) (string, error) {
		expr, _ := params["expression"].(string)
		if expr == "" {
			return "", fmt.Errorf("expression is required")
		}
		// Simple eval - for production use a proper math parser
		return fmt.Sprintf("Result: %s", expr), nil
	}
}

func (m *SkillManager) ListSkills() []Skill {
	var result []Skill
	for name := range m.skills {
		isPython := false
		if m.pythonDir != "" {
			scriptPath := filepath.Join(m.pythonDir, name, name+".py")
			if _, err := os.Stat(scriptPath); err == nil {
				isPython = true
			}
		}

		desc := fmt.Sprintf("Built-in skill: %s", name)
		if isPython {
			if meta, ok := m.skillMeta[name]; ok {
				desc = meta.Description
			} else {
				desc = fmt.Sprintf("Python skill: %s", name)
			}
		}

		result = append(result, Skill{
			Name:        name,
			Description: desc,
			IsPython:    isPython,
		})
	}
	return result
}

func (m *SkillManager) GetSkillSummaries() string {
	var summaries []string
	for name := range m.skills {
		desc := fmt.Sprintf("- %s: %s", name, m.getSkillDescription(name))
		summaries = append(summaries, desc)
	}
	return "Available Skills:\n" + strings.Join(summaries, "\n")
}

func (m *SkillManager) getSkillDescription(name string) string {
	if m.pythonDir != "" {
		if meta, ok := m.skillMeta[name]; ok {
			return meta.Description
		}
	}
	return fmt.Sprintf("Built-in skill: %s", name)
}

func (m *SkillManager) ExecuteSkill(name string, params map[string]interface{}) (string, error) {
	fn, ok := m.skills[name]
	if !ok {
		if m.pythonDir != "" {
			return m.executePython(name, params)
		}
		return "", fmt.Errorf("skill not found: %s", name)
	}
	return fn(context.Background(), params)
}

func (m *SkillManager) AddSkill(name string, fn SkillFunc) {
	m.skills[name] = fn
}

func (m *SkillManager) executePython(name string, params map[string]interface{}) (string, error) {
	scriptPath := filepath.Join(m.pythonDir, name, name+".py")
	if _, err := os.Stat(scriptPath); err != nil {
		return "", fmt.Errorf("python skill not found: %s", name)
	}

	pythonCmds := []string{"py", "python", "python3"}
	var lastErr error
	var output []byte

	paramsJson, _ := json.Marshal(params)

	for _, pythonCmd := range pythonCmds {
		cmd := exec.Command(pythonCmd, scriptPath, string(paramsJson))
		output, err := cmd.CombinedOutput()
		if err == nil {
			return string(output), nil
		}
		lastErr = err
	}

	return string(output), lastErr
}

func (m *SkillManager) RegisterPythonSkill(name string) error {
	m.skills[name] = func(ctx context.Context, params map[string]interface{}) (string, error) {
		return m.executePython(name, params)
	}
	return nil
}
