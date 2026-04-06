package tools

import (
	"sort"
	"strings"
)

// Registry 定义工具注册、查找和列举的统一接口。
type Registry interface {
	Register(tool Tool) error
	Get(name string) (Tool, bool)
	List() []ToolSpec
}

// InMemoryRegistry 使用内存 map 保存已注册工具。
type InMemoryRegistry struct {
	tools map[string]Tool
}

// NewInMemoryRegistry 创建一个新的内存工具注册表。
func NewInMemoryRegistry() *InMemoryRegistry {
	return &InMemoryRegistry{
		tools: make(map[string]Tool),
	}
}

// Register 将一个工具注册到当前注册表中。
func (r *InMemoryRegistry) Register(tool Tool) error {
	spec := tool.Spec()
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return ErrInvalidToolInput
	}
	if _, exists := r.tools[name]; exists {
		return ErrDuplicateTool
	}

	r.tools[name] = tool
	return nil
}

// Get 按工具名称查找已注册工具。
func (r *InMemoryRegistry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[strings.TrimSpace(name)]
	return tool, ok
}

// List 返回当前已注册工具的描述列表，并按名称稳定排序。
func (r *InMemoryRegistry) List() []ToolSpec {
	names := sortedToolNames(r.tools)
	specs := make([]ToolSpec, 0, len(names))
	for _, name := range names {
		specs = append(specs, r.tools[name].Spec())
	}

	return specs
}

// sortedToolNames 返回按字典序排序后的工具名称列表。
func sortedToolNames(tools map[string]Tool) []string {
	names := make([]string, 0, len(tools))
	for name := range tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
