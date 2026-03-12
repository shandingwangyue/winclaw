package system

type PermissionLevel string

const (
	PermissionLow    PermissionLevel = "low"
	PermissionMedium PermissionLevel = "medium"
	PermissionHigh   PermissionLevel = "high"
	PermissionFull   PermissionLevel = "full"
)

type ConfirmMode string

const (
	ConfirmAlways ConfirmMode = "always"
	ConfirmFirst  ConfirmMode = "first"
	ConfirmAuto   ConfirmMode = "auto"
)

type Permission struct {
	Level       PermissionLevel `json:"level"`
	Whitelist   []string        `json:"whitelist"`
	Blacklist   []string        `json:"blacklist"`
	ConfirmMode ConfirmMode     `json:"confirm_mode"`
}

var defaultPermission = Permission{
	Level:       PermissionMedium,
	ConfirmMode: ConfirmFirst,
}

type PermissionManager struct {
	perm Permission
}

func NewPermissionManager() *PermissionManager {
	return &PermissionManager{perm: defaultPermission}
}

func (p *PermissionManager) SetPermission(perm Permission) {
	p.perm = perm
}

func (p *PermissionManager) GetPermission() Permission {
	return p.perm
}

func (p *PermissionManager) CanExecute(operation string) bool {
	for _, op := range p.perm.Blacklist {
		if op == operation {
			return false
		}
	}
	if p.perm.Level == PermissionFull {
		return true
	}
	if p.perm.Level == PermissionLow {
		return false
	}
	return true
}

func (p *PermissionManager) NeedConfirm(operation string) bool {
	if p.perm.ConfirmMode == ConfirmAuto {
		return false
	}
	if p.perm.ConfirmMode == ConfirmFirst {
		return true
	}
	return p.perm.Level != PermissionFull
}

func (p *PermissionManager) IsAllowedOperation(op string) bool {
	switch op {
	case "read_file", "list_dir", "system_info":
		return p.perm.Level != PermissionLow
	case "run_program", "open_browser", "clipboard", "scan_apps":
		return p.perm.Level == PermissionMedium || p.perm.Level == PermissionHigh || p.perm.Level == PermissionFull
	case "write_file", "delete_file", "word_doc", "excel_doc":
		return p.perm.Level == PermissionHigh || p.perm.Level == PermissionFull
	case "screenshot":
		return p.perm.Level != PermissionLow
	default:
		return p.perm.Level == PermissionFull
	}
}
