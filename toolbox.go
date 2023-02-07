package trails

// A Toolbox is a set of Tools exposed to the end user
// in certain environments, notably, not in Production.
// Generally, these are administrative tools that
// simplify demonstrating features
// which would otherwise require actions taken in many steps.
type Toolbox []Tool

// Filter returns a Toolbox after removing all Tools that cannot be rendered.
// If none can be rendered, Filter returns a zero-value Toolbox.
func (t Toolbox) Filter() Toolbox {
	var n int
	for _, tool := range t {
		if tool.Render() {
			t[n] = tool
			n++
		}
	}

	if n == 0 {
		return make(Toolbox, 0)
	}

	return t[:n]
}

// A Tool is a set of actions grouped under a category.
// A Tool may pertain to a part of the domain,
// grouping actions touching similar models.
type Tool struct {
	Actions []ToolAction `json:"actions"`
	Title   string       `json:"title"`
}

// Render asserts whether the Tool should be rendered.
func (t Tool) Render() bool { return len(t.Actions) > 0 }

// A ToolAction is a specific link the end user can follow
// to execute the named action.
type ToolAction struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}
