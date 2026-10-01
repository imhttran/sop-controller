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

// AttentionIntervalDefault is the documented short default cadence at which the
// live-progress transports re-read SOP's reported human-decision gate, so a
// newly recorded gate becomes visible promptly (a small multiple of this
// interval) rather than after an arbitrary long wait. It is deliberately short
// and there is no multi-minute fallback anywhere.
const AttentionIntervalDefault = time.Second

// AttentionIntervalFloor is the lower bound the resolved attention cadence is
// clamped to. A configured value below it is raised to it, so the cadence can
// never be zero/negative (which would busy-poll) while still guaranteeing
// promptness.
const AttentionIntervalFloor = 250 * time.Millisecond

// AttentionPollEnv is the environment variable that configures the short
// attention-poll cadence.
const AttentionPollEnv = "SOP_CONTROLLER_ATTENTION_POLL"

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
	// AttentionInterval is the short, configurable cadence at which the
	// live-progress transports re-read SOP's reported human-decision gate so a
	// newly recorded gate surfaces promptly. It governs only how often SOP is
	// re-read; it is never used to infer a gate from inactivity, and it is never
	// a fixed multi-minute wait.
	AttentionInterval time.Duration
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
		Addr:              envOr("SOP_CONTROLLER_ADDR", "127.0.0.1:8080"),
		ProjectRoots:      projectRoots,
		Workspaces:        splitList(workspaceEnv),
		DiscoveryDepth:    intOr("SOP_CONTROLLER_DISCOVERY_DEPTH", DefaultMaxDepth),
		SOPBin:            envOr("SOP_BIN", "sop"),
		CommandTimeout:    durationOr("SOP_CONTROLLER_COMMAND_TIMEOUT", 15*time.Minute),
		PollInterval:      durationOr("SOP_CONTROLLER_POLL", 3*time.Second),
		AttentionInterval: ResolveAttentionInterval(os.Getenv(AttentionPollEnv)),
		AllowNetwork:      boolOr("SOP_CONTROLLER_ALLOW_NETWORK", false),
		AccessToken:       os.Getenv("SOP_CONTROLLER_TOKEN"),
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

// ResolveAttentionInterval turns the raw attention-poll setting into the
// resolved short attention cadence.
//
// Contract:
//   - empty or unparseable input resolves to AttentionIntervalDefault (the
//     documented short default), never to a long wait;
//   - a positive input is clamped up to AttentionIntervalFloor so the cadence is
//     always a bounded, non-zero short interval;
//   - the result is never a fixed multi-minute wait.
//
// The cadence governs only how often the existing live-progress transports
// re-read SOP's reported gate. It is never consulted to infer a gate from
// inactivity.
func ResolveAttentionInterval(raw string) time.Duration {
	d, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil || d <= 0 {
		return AttentionIntervalDefault
	}
	if d < AttentionIntervalFloor {
		return AttentionIntervalFloor
	}
	return d
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
