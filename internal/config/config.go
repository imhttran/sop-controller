// Package config holds runtime configuration for the SOP Controller dashboard.
package config

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is read from the environment (optionally seeded from .env / .env.dev).
type Config struct {
	// Addr is the HTTP listen address. Loopback by default.
	Addr string
	// ProjectRoots are directories that each contain a .agent-sdlc/ SOP state.
	ProjectRoots []string
	// Workspaces are directories whose SOP projects are discovered by scanning
	// for .agent-sdlc/config.yaml. They are an explicit allowlist, never an
	// implicit filesystem crawl.
	Workspaces []string
	// DiscoveryDepth bounds how far below a workspace root discovery descends.
	DiscoveryDepth int
	// SOPBin is the SOP CLI used for commands (run/resume/validate/review).
	SOPBin string
	// CommandTimeout bounds a single SOP command.
	CommandTimeout time.Duration
	// PollInterval is the default HTMX poll cadence for active views.
	PollInterval time.Duration
	// AllowNetwork permits binding to a non-loopback address (requires AccessToken).
	AllowNetwork bool
	// AccessToken gates access when AllowNetwork is on.
	AccessToken string
}

// LoadEnvFiles mirrors the previous loader: a personal .env wins (existing env
// vars are never overwritten); .env.dev only fills in for development.
func LoadEnvFiles() {
	applyEnv := func(path string) {
		f, err := os.Open(path)
		if err != nil {
			return
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			line = strings.TrimPrefix(line, "export ")
			key, value, found := strings.Cut(line, "=")
			if !found {
				continue
			}
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"' ||
				value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
			if _, set := os.LookupEnv(key); !set {
				os.Setenv(key, value)
			}
		}
	}
	applyEnv(".env")
	if env := os.Getenv("SOP_CONTROLLER_ENV"); env == "" || env == "development" {
		if env := os.Getenv("NODE_ENV"); env == "" || env == "development" {
			applyEnv(".env.dev")
		}
	}
}

// Load reads configuration from the environment after LoadEnvFiles.
func Load() Config {
	projectEnv := strings.TrimSpace(os.Getenv("SOP_CONTROLLER_PROJECTS"))
	workspaceEnv := strings.TrimSpace(os.Getenv("SOP_CONTROLLER_WORKSPACES"))
	// Explicit projects stay backward compatible: when neither variable is set
	// the controller observes the current directory. When workspace roots are
	// configured they are sufficient on their own, so the "." default is not
	// forced (which would otherwise fail when the controller runs outside a
	// project).
	projectRoots := splitList(projectEnv)
	if projectEnv == "" && workspaceEnv == "" {
		projectRoots = []string{"."}
	}
	c := Config{
		Addr:           envOr("SOP_CONTROLLER_ADDR", "127.0.0.1:8080"),
		ProjectRoots:   projectRoots,
		Workspaces:     splitList(workspaceEnv),
		DiscoveryDepth: intOr("SOP_CONTROLLER_DISCOVERY_DEPTH", DefaultMaxDepth),
		SOPBin:         envOr("SOP_BIN", "sop"),
		CommandTimeout: durationOr("SOP_CONTROLLER_COMMAND_TIMEOUT", 15*time.Minute),
		PollInterval:   durationOr("SOP_CONTROLLER_POLL", 3*time.Second),
		AllowNetwork:   boolOr("SOP_CONTROLLER_ALLOW_NETWORK", false),
		AccessToken:    os.Getenv("SOP_CONTROLLER_TOKEN"),
	}
	if !c.AllowNetwork && !isLoopback(c.Addr) {
		log.Printf("[config] %q is not loopback; restricting to 127.0.0.1 (set SOP_CONTROLLER_ALLOW_NETWORK=true to allow)", c.Addr)
		c.Addr = "127.0.0.1" + portSuffix(c.Addr)
	}
	if c.AllowNetwork && c.AccessToken == "" {
		log.Fatal("[config] SOP_CONTROLLER_ALLOW_NETWORK=true requires SOP_CONTROLLER_TOKEN")
	}
	return c
}

func isLoopback(addr string) bool {
	host := addr
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		host = addr[:i]
	}
	return host == "127.0.0.1" || host == "localhost" || host == "::1" || host == ""
}

func portSuffix(addr string) string {
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		return addr[i:]
	}
	return ":8080"
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func boolOr(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func durationOr(key string, fallback time.Duration) time.Duration {
	if d, err := time.ParseDuration(os.Getenv(key)); err == nil && d > 0 {
		return d
	}
	return fallback
}

// intOr reads a positive integer, falling back on empty, unparseable, or
// non-positive values (a discovery depth of zero or less is meaningless).
func intOr(key string, fallback int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v > 0 {
		return v
	}
	return fallback
}

// splitList parses a comma-separated path list, dropping blank entries. It never
// substitutes a default: callers decide whether an empty list means "none" or
// "the current directory".
func splitList(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
