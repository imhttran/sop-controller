// Command sop-controller serves the local-first SOP dashboard.
package main

import (
	"io/fs"
	"log"
	"net/http"

	assets "sop-controller"

	"sop-controller/internal/config"
	"sop-controller/internal/sopclient"
	"sop-controller/internal/web"
)

func main() {
	config.LoadEnvFiles()
	cfg := config.Load()

	// Merge explicit project roots with projects discovered beneath the configured
	// workspace roots. Discovery is an allowlist: it only looks under
	// SOP_CONTROLLER_WORKSPACES, never the wider filesystem.
	report, err := config.ResolveReport(cfg.ProjectRoots, cfg.Workspaces, cfg.DiscoveryDepth)
	if err != nil {
		log.Fatalf("project discovery: %v", err)
	}
	for _, d := range report.Diagnostics {
		log.Printf("[discovery] %s: %s", d.Path, d.Reason)
	}
	roots := make([]string, 0, len(report.Projects))
	for _, p := range report.Projects {
		roots = append(roots, p.Root)
	}
	sop, err := sopclient.New(roots, cfg.SOPBin, cfg.CommandTimeout)
	if err != nil {
		log.Fatalf("SOP boundary: %v", err)
	}
	defer sop.Close()

	views, err := web.NewViews(assets.FS)
	if err != nil {
		log.Fatalf("parse templates: %v", err)
	}
	staticFS, err := fs.Sub(assets.FS, "static")
	if err != nil {
		log.Fatalf("static assets: %v", err)
	}

	handler := web.NewServer(web.Options{
		SOP:            sop,
		Views:          views,
		StaticFS:       staticFS,
		Poll:           cfg.PollInterval,
		CommandTimeout: cfg.CommandTimeout,
		AllowNetwork:   cfg.AllowNetwork,
		AccessToken:    cfg.AccessToken,
		Discovery:      report,
	})

	log.Printf("SOP Controller listening on http://%s (%d project(s), poll %s)", cfg.Addr, len(roots), cfg.PollInterval)
	if err := http.ListenAndServe(cfg.Addr, handler); err != nil {
		log.Fatal(err)
	}
}
