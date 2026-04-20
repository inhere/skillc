package agent

type Scope string

const (
	ScopeUser    Scope = "user"
	ScopeProject Scope = "project"
)

const DefaultAgentName = "universal"
