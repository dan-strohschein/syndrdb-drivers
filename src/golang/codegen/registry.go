package codegen

import (
	"sync"

	"github.com/dan-strohschein/syndrdb-drivers/src/golang/schema"
)

// TypeRegistry caches type information for code generation.
type TypeRegistry struct {
	bundles map[string]*schema.BundleDefinition
	mu      sync.RWMutex
}

// NewTypeRegistry creates a new type registry.
func NewTypeRegistry() *TypeRegistry {
	return &TypeRegistry{
		bundles: make(map[string]*schema.BundleDefinition),
	}
}

// Register adds a bundle definition to the registry.
func (r *TypeRegistry) Register(bundle *schema.BundleDefinition) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bundles[bundle.Name] = bundle
}

// Get retrieves a bundle definition by name.
func (r *TypeRegistry) Get(name string) (*schema.BundleDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	bundle, exists := r.bundles[name]
	return bundle, exists
}

// GetAll returns all registered bundle definitions.
func (r *TypeRegistry) GetAll() []*schema.BundleDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	bundles := make([]*schema.BundleDefinition, 0, len(r.bundles))
	for _, bundle := range r.bundles {
		bundles = append(bundles, bundle)
	}
	return bundles
}

// Clear removes all entries from the registry.
func (r *TypeRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bundles = make(map[string]*schema.BundleDefinition)
}

// LoadFromSchema populates the registry from a schema definition.
func (r *TypeRegistry) LoadFromSchema(schemaDef *schema.SchemaDefinition) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range schemaDef.Bundles {
		r.bundles[schemaDef.Bundles[i].Name] = &schemaDef.Bundles[i]
	}
}

// Count returns the number of registered bundles.
func (r *TypeRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.bundles)
}

// Has checks if a bundle is registered.
func (r *TypeRegistry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.bundles[name]
	return exists
}
