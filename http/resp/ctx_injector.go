package resp

import "context"

// ContextInjector is the interface for describing how values from context.Context can be
// merged with existing keys in a map[string]interface{}.
type ContextInjector interface {
	Inject(props map[string]interface{}, ctx context.Context)
}

// A DefaultInjector holds the keys required to pull values from a context.Context.
//
// DefaultInjector implements ContextInjector
type DefaultInjector struct {
	Keys []string
}

// Inject merges into props the key-value pairs pulled from ctx using i.Keys
// if the value is for a certain key is not null.
func (i DefaultInjector) Inject(props map[string]interface{}, ctx context.Context) {
	if props == nil || ctx == nil || i.Keys == nil {
		return
	}
	for _, k := range i.Keys {
		if val := ctx.Value(k); val != nil {
			props[k] = val
		}
	}
}

// A NoopInjector implements ContextInjector and performs no operation.
type NoopInjector struct{}

func (NoopInjector) Inject(_ map[string]interface{}, _ context.Context) {}
