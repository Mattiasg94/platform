package tools

import (
	"context"
	"fmt"

	"orchestrator/internal/activities"
)

type Registry struct {
	sb activities.Sandbox
}

func NewRegistry(sb activities.Sandbox) *Registry {
	return &Registry{sb: sb}
}

func (r *Registry) Execute(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	switch name {
	case "read_file":
		return r.executeReadFile(ctx, args)
	case "write_file":
		return r.executeWriteFile(ctx, args)
	case "bash":
		return r.executeBash(ctx, args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (r *Registry) executeReadFile(ctx context.Context, args map[string]any) (map[string]any, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("path argument must be a non-empty string")
	}
	out, err := activities.Read(ctx, r.sb, activities.ReadInput{Path: path})
	if err != nil {
		return nil, err
	}
	return map[string]any{"content": out.Content}, nil
}

func (r *Registry) executeWriteFile(ctx context.Context, args map[string]any) (map[string]any, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("path argument must be a non-empty string")
	}
	content, ok := args["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content argument must be a string")
	}
	if _, err := activities.Write(ctx, r.sb, activities.WriteInput{Path: path, Content: content}); err != nil {
		return nil, err
	}
	return map[string]any{"status": "ok"}, nil
}

func (r *Registry) executeBash(ctx context.Context, args map[string]any) (map[string]any, error) {
	rawCmd, ok := args["command"].([]any)
	if !ok || len(rawCmd) == 0 {
		return nil, fmt.Errorf("command argument must be a non-empty array of strings")
	}
	cmd := make([]string, len(rawCmd))
	for i, v := range rawCmd {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("command[%d] must be a string", i)
		}
		cmd[i] = s
	}
	dir, _ := args["dir"].(string)

	out, err := activities.Exec(ctx, r.sb, activities.ExecInput{Command: cmd, Dir: dir})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"stdout":    out.Stdout,
		"stderr":    out.Stderr,
		"exit_code": out.ExitCode,
	}, nil
}
