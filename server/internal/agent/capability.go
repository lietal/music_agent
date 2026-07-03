package agent

type CapabilityManifest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tools       []string `json:"tools"`
	Inputs      []string `json:"inputs,omitempty"`
	Outputs     []string `json:"outputs,omitempty"`
}

type CapabilityRegistry struct {
	capabilities map[string]CapabilityManifest
}

func NewCapabilityRegistry() *CapabilityRegistry {
	return &CapabilityRegistry{capabilities: make(map[string]CapabilityManifest)}
}

func (r *CapabilityRegistry) Register(m CapabilityManifest) {
	r.capabilities[m.Name] = m
}

func (r *CapabilityRegistry) Get(name string) (CapabilityManifest, bool) {
	m, ok := r.capabilities[name]
	return m, ok
}

func (r *CapabilityRegistry) List() []CapabilityManifest {
	result := make([]CapabilityManifest, 0, len(r.capabilities))
	for _, m := range r.capabilities {
		result = append(result, m)
	}
	return result
}
