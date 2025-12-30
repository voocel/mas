package policy

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/voocel/mas/executor"
	"github.com/voocel/mas/tools"
)

func ValidateToolPolicy(policy executor.Policy, tool tools.Tool, args json.RawMessage) error {
	if !isToolAllowed(policy, tool) {
		return errors.New("tool not allowed")
	}
	if !policy.Network.Enabled && hasCapability(tool, tools.CapabilityNetwork) {
		return errors.New("network disabled")
	}
	if tool.Name() == "file_system" {
		return validateFileAccess(policy, args)
	}
	if hasCapability(tool, tools.CapabilityNetwork) && len(policy.Network.AllowedHosts) > 0 {
		if err := validateURLHosts(policy.Network.AllowedHosts, args); err != nil {
			return err
		}
	}
	return nil
}

func isToolAllowed(policy executor.Policy, tool tools.Tool) bool {
	if len(policy.AllowedTools) > 0 {
		allowed := false
		for _, name := range policy.AllowedTools {
			if name == tool.Name() {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}
	if len(policy.AllowedCaps) == 0 {
		return true
	}

	caps := tool.Capabilities()
	if len(caps) == 0 {
		return false
	}

	allowedCaps := make(map[tools.Capability]struct{}, len(policy.AllowedCaps))
	for _, cap := range policy.AllowedCaps {
		allowedCaps[cap] = struct{}{}
	}

	for _, cap := range caps {
		if _, ok := allowedCaps[cap]; !ok {
			return false
		}
	}
	return true
}

func hasCapability(tool tools.Tool, cap tools.Capability) bool {
	for _, c := range tool.Capabilities() {
		if c == cap {
			return true
		}
	}
	return false
}

func validateFileAccess(policy executor.Policy, args json.RawMessage) error {
	path := extractPath(args)
	if path == "" {
		return errors.New("path is required")
	}

	allowed := append([]string{}, policy.AllowedPaths...)
	if policy.Workdir != "" {
		allowed = append(allowed, policy.Workdir)
	}
	if len(allowed) == 0 {
		return errors.New("path not allowed")
	}

	absPath, err := resolvePath(path)
	if err != nil {
		return errors.New("invalid path")
	}

	for _, p := range allowed {
		absAllowed, err := resolvePath(p)
		if err != nil {
			continue
		}
		if absPath == absAllowed || strings.HasPrefix(absPath, absAllowed+string(filepath.Separator)) {
			return nil
		}
	}
	return errors.New("path not allowed")
}

func resolvePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	absPath = filepath.Clean(absPath)

	if resolved, err := filepath.EvalSymlinks(absPath); err == nil {
		return resolved, nil
	}

	dir := absPath
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		if _, statErr := os.Stat(dir); statErr == nil {
			resolved, err := filepath.EvalSymlinks(dir)
			if err == nil {
				rel, relErr := filepath.Rel(dir, absPath)
				if relErr == nil && rel != "." {
					return filepath.Join(resolved, rel), nil
				}
				return resolved, nil
			}
		}
		dir = parent
	}
	return absPath, nil
}

func validateURLHosts(allowed []string, args json.RawMessage) error {
	rawURL := extractURL(args)
	if rawURL == "" {
		return nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Hostname() == "" {
		return errors.New("invalid url")
	}
	host := parsed.Hostname()
	for _, allowedHost := range allowed {
		if host == allowedHost {
			return nil
		}
	}
	return errors.New("host not allowed")
}

func extractPath(args json.RawMessage) string {
	var payload map[string]any
	if err := json.Unmarshal(args, &payload); err != nil {
		return ""
	}
	if value, ok := payload["path"].(string); ok {
		return value
	}
	return ""
}

func extractURL(args json.RawMessage) string {
	var payload map[string]any
	if err := json.Unmarshal(args, &payload); err != nil {
		return ""
	}
	if value, ok := payload["url"].(string); ok {
		return value
	}
	return ""
}
