package resource

import "sort"

// Registry holds all registered resource kinds.
type Registry struct {
	resources map[string]Resource
}

// NewRegistry creates an empty resource registry.
func NewRegistry() *Registry {
	return &Registry{resources: make(map[string]Resource)}
}

// Register adds a resource to the registry, keyed by its Schema().Kind.
func (r *Registry) Register(res Resource) {
	r.resources[res.Schema().Kind] = res
}

// Get retrieves a resource by kind name.
func (r *Registry) Get(kind string) (Resource, bool) {
	res, ok := r.resources[kind]
	return res, ok
}

// All returns every registered resource, sorted alphabetically by kind.
func (r *Registry) All() []Resource {
	keys := make([]string, 0, len(r.resources))
	for k := range r.resources {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	result := make([]Resource, len(keys))
	for i, k := range keys {
		result[i] = r.resources[k]
	}
	return result
}
