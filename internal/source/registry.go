package source

import "fmt"

type Registry struct{ plugins map[string]*Plugin }

func NewRegistry() *Registry { return &Registry{plugins: map[string]*Plugin{}} }
func (r *Registry) Add(p *Plugin) error {
	if _, ok := r.plugins[p.Metadata.ID]; ok {
		return fmt.Errorf("duplicate source %q", p.Metadata.ID)
	}
	r.plugins[p.Metadata.ID] = p
	return nil
}
func (r *Registry) Get(id string) (*Plugin, bool) { p, ok := r.plugins[id]; return p, ok }
