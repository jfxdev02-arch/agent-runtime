package tools

import "fmt"

type ToolContext struct {
	SessionID string
	Workspace string
	Depth     int
	MaxDepth  int
}

type ToolParam struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

type Tool interface {
	Name() string
	Description() string
	Risk() string // "LOW", "HIGH"
	Parameters() []ToolParam
	Execute(ctx ToolContext, args map[string]string) (string, error)
}

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

func (r *Registry) Get(name string) (Tool, error) {
	if t, ok := r.tools[name]; ok {
		return t, nil
	}
	return nil, fmt.Errorf("tool %s not found", name)
}

func (r *Registry) ListTools() []Tool {
	list := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		list = append(list, t)
	}
	return list
}

func (r *Registry) FilterByNames(names []string) *Registry {
	filtered := NewRegistry()
	for _, name := range names {
		if t, ok := r.tools[name]; ok {
			filtered.Register(t)
		}
	}
	return filtered
}

func (r *Registry) Clone() *Registry {
	clone := NewRegistry()
	for name, t := range r.tools {
		clone.tools[name] = t
	}
	return clone
}
