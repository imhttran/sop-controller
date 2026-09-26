package config

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Project is a resolved SOP project candidate: a canonical project root plus the
// controller's identity for it.
type Project struct {
	// Root is the canonical (absolute, symlink-resolved) project directory.
	Root string
	// ID is the controller-facing project identity used in routes and stores.
	ID string
}

// Diagnostic reports a candidate the controller skipped during discovery, or a
// workspace root it could not read. Path and Reason are safe to log: neither is a
// configuration value or secret.
type Diagnostic struct {
	Path   string
	Reason string
}

// DefaultMaxDepth bounds how far below a workspace root discovery descends.
// workspace/project is depth 1; workspace/group/project is depth 2.
const DefaultMaxDepth = 4

// ignoredDirs are directory names discovery never descends into: VCS metadata,
// dependency trees, build output, and caches. None can hold a project's
// .agent-sdlc/config.yaml, and walking them would be expensive and pointless.
var ignoredDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	"target":       true,
	".cache":       true,
}

// ProjectConfig is the small, non-secret subset of .agent-sdlc/config.yaml the
// controller reads.
type ProjectConfig struct {
	Name   string
	Branch string
}

// ReadProjectConfig parses .agent-sdlc/config.yaml under root. It is the single
// place that interprets SOP project configuration, so discovery validation and
// the SOP state store agree on what a project is. The parser is intentionally
// minimal (no YAML dependency): it recognizes the top-level `project` section and
// reads `name` and `integration_branch`. A file with no `project` section is
// rejected as invalid.
func ReadProjectConfig(root string) (ProjectConfig, error) {
	path := filepath.Join(root, ".agent-sdlc", "config.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		return ProjectConfig{}, err
	}
	var cfg ProjectConfig
	hasProject := false
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		switch key {
		case "project":
			hasProject = true
		case "name":
			if cfg.Name == "" {
				cfg.Name = value
			}
		case "integration_branch":
			cfg.Branch = value
		}
	}
	if !hasProject {
		return ProjectConfig{}, fmt.Errorf("%s: missing project section", path)
	}
	return cfg, nil
}

// ProjectID is the controller's stable identity for a project. It prefers the
// project's declared name from .agent-sdlc/config.yaml and falls back to the root
// directory's name. Discovery and the SOP state store both use it, so a project
// has one identity however it was configured or discovered.
func ProjectID(root string) string {
	if cfg, err := ReadProjectConfig(root); err == nil && cfg.Name != "" {
		return cfg.Name
	}
	return filepath.Base(root)
}

// DiscoverProjects finds SOP projects beneath the given workspace roots. A
// directory is a project when it contains a valid .agent-sdlc/config.yaml and an
// initialized .agent-sdlc/state.db. Discovery is bounded to maxDepth (0 uses
// DefaultMaxDepth), prunes ignoredDirs, and follows no symlinks: filepath.WalkDir
// lstats entries, so a symlinked directory is never descended into. It therefore
// cannot loop and cannot escape a configured workspace root.
//
// A directory that is not a project yields nothing; a directory that looks like a
// project but has an invalid config or missing state yields a Diagnostic and is
// skipped, so one bad directory never hides unrelated valid projects.
func DiscoverProjects(workspaces []string, maxDepth int) ([]Project, []Diagnostic) {
	if maxDepth <= 0 {
		maxDepth = DefaultMaxDepth
	}
	var projects []Project
	var diags []Diagnostic
	for _, ws := range workspaces {
		root, err := canonicalPath(ws)
		if err != nil {
			diags = append(diags, Diagnostic{Path: ws, Reason: err.Error()})
			continue
		}
		info, err := os.Stat(root)
		if err != nil {
			diags = append(diags, Diagnostic{Path: ws, Reason: "workspace root not readable: " + err.Error()})
			continue
		}
		if !info.IsDir() {
			diags = append(diags, Diagnostic{Path: ws, Reason: "workspace root is not a directory"})
			continue
		}
		walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				// One unreadable entry must not abort the rest of the walk.
				diags = append(diags, Diagnostic{Path: path, Reason: "not readable: " + err.Error()})
				return nil
			}
			if !d.IsDir() {
				return nil
			}
			if path != root {
				if ignoredDirs[d.Name()] || depth(root, path) > maxDepth {
					return fs.SkipDir
				}
			}
			project, isProject, diag := considerProject(path)
			if diag != "" {
				diags = append(diags, Diagnostic{Path: path, Reason: diag})
				return nil
			}
			if isProject {
				projects = append(projects, project)
				// A project does not contain nested projects; do not descend.
				return fs.SkipDir
			}
			return nil
		})
		if walkErr != nil {
			diags = append(diags, Diagnostic{Path: root, Reason: "walk failed: " + walkErr.Error()})
		}
	}
	return projects, diags
}

// considerProject classifies a directory. It returns (project, true, "") for a
// valid SOP project, ("", false, "") for an ordinary directory, and
// ("", false, reason) for a directory that carries a .agent-sdlc/config.yaml but
// is not usable (invalid config, or no initialized state database).
func considerProject(dir string) (Project, bool, string) {
	if _, err := os.Stat(filepath.Join(dir, ".agent-sdlc", "config.yaml")); err != nil {
		return Project{}, false, ""
	}
	cfg, err := ReadProjectConfig(dir)
	if err != nil {
		return Project{}, false, "invalid SOP config: " + err.Error()
	}
	if _, err := os.Stat(filepath.Join(dir, ".agent-sdlc", "state.db")); err != nil {
		return Project{}, false, "SOP config present but no initialized state.db"
	}
	id := cfg.Name
	if id == "" {
		id = filepath.Base(dir)
	}
	return Project{Root: dir, ID: id}, true, ""
}

// ResolveProjects merges explicitly configured project roots with projects
// discovered beneath the configured workspace roots. Roots are canonicalized and
// deduplicated, so a project reachable both ways appears once; explicit roots are
// added first, in configured order, then discovered roots in walk order. maxDepth
// bounds discovery (0 uses DefaultMaxDepth).
//
// Two distinct roots that claim the same project identity are a deterministic
// conflict: ResolveProjects returns an error naming both roots rather than bind
// the identity to either, so the controller can never silently expose the wrong
// repository.
func ResolveProjects(explicit, workspaces []string, maxDepth int) ([]Project, []Diagnostic, error) {
	var out []Project
	var diags []Diagnostic
	seenRoot := map[string]bool{}
	rootByID := map[string]string{}

	add := func(root, id string) error {
		if seenRoot[root] {
			return nil
		}
		if prev, dup := rootByID[id]; dup {
			return fmt.Errorf("project identity %q is claimed by two roots: %s and %s", id, prev, root)
		}
		seenRoot[root] = true
		rootByID[id] = root
		out = append(out, Project{Root: root, ID: id})
		return nil
	}

	for _, r := range explicit {
		root, err := canonicalPath(r)
		if err != nil {
			diags = append(diags, Diagnostic{Path: r, Reason: err.Error()})
			continue
		}
		if err := add(root, ProjectID(root)); err != nil {
			return nil, diags, err
		}
	}

	discovered, discoveryDiags := DiscoverProjects(workspaces, maxDepth)
	diags = append(diags, discoveryDiags...)
	for _, p := range discovered {
		if err := add(p.Root, p.ID); err != nil {
			return nil, diags, err
		}
	}
	return out, diags, nil
}

// DiscoveryReport is a read-only snapshot of how the controller resolved its
// project registry: the configured explicit roots and workspace roots, the depth
// bound applied, the projects registered, and the candidates skipped. It is
// controller diagnostics, never SOP state.
type DiscoveryReport struct {
	Explicit    []string
	Workspaces  []string
	MaxDepth    int
	Projects    []Project
	Diagnostics []Diagnostic
}

// ResolveReport resolves the registry (see ResolveProjects) and returns a full
// report. On an identity conflict the report carries no projects and the error is
// returned so the caller can surface it.
func ResolveReport(explicit, workspaces []string, maxDepth int) (DiscoveryReport, error) {
	projects, diags, err := ResolveProjects(explicit, workspaces, maxDepth)
	rep := DiscoveryReport{
		Explicit:    explicit,
		Workspaces:  workspaces,
		MaxDepth:    maxDepth,
		Projects:    projects,
		Diagnostics: diags,
	}
	if err != nil {
		rep.Projects = nil
		return rep, err
	}
	return rep, nil
}

// canonicalPath resolves p to an absolute, symlink-resolved, cleaned path. It is
// the dedupe key for a project root: the same directory reached two ways (explicit
// and discovered, or through a symlink) collapses to one entry.
func canonicalPath(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved, nil
	}
	return filepath.Clean(abs), nil
}

// depth is the number of path segments from a workspace root to a directory:
// root itself is 0, workspace/project is 1, workspace/group/project is 2.
func depth(root, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	return strings.Count(rel, string(filepath.Separator)) + 1
}
