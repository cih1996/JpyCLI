package config

// LocalServerConfig represents the CLI configuration for a server
type LocalServerConfig struct {
	URL            string `json:"url" yaml:"url"`
	Username       string `json:"username" yaml:"username"`
	Password       string `json:"password" yaml:"password"`
	Group          string `json:"group" yaml:"group"` // Customer/Tenant Group
	Token          string `json:"token,omitempty" yaml:"token,omitempty"`
	LastLoginTime  string `json:"last_login_time,omitempty" yaml:"last_login_time,omitempty"`
	LastLoginError string `json:"last_login_error,omitempty" yaml:"last_login_error,omitempty"`
	Disabled       bool   `json:"disabled,omitempty" yaml:"disabled,omitempty"` // Soft delete
}

// BatchItem represents an item for batch operations
type BatchItem struct {
	URL   string `json:"url"`
	Group string `json:"group"`
}

// Config represents the CLI configuration file structure
type Config struct {
	// Deprecated: Use Groups instead. Kept for migration.
	Servers        []LocalServerConfig            `json:"servers,omitempty" yaml:"servers,omitempty"`
	Groups         map[string][]LocalServerConfig `json:"groups" yaml:"groups"`
	ActiveGroup    string                         `json:"active_group" yaml:"active_group"`
	ActiveContext  string                         `json:"active_context" yaml:"active_context"` // Usually URL
	Admin          *AdminConfig                   `json:"admin,omitempty" yaml:"admin,omitempty"`
	AdminAuth      *AdminConfig                   `json:"admin-auth,omitempty" yaml:"admin-auth,omitempty"`
	AdminOperation *AdminConfig                   `json:"admin-operation,omitempty" yaml:"admin-operation,omitempty"`
}

type AdminConfig struct {
	Token    string `json:"token" yaml:"token"`
	Username string `json:"username" yaml:"username"`
}
