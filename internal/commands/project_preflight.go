package commands

import (
	"fmt"
	"strings"
	"time"
)

func runProjectPreflight(path string, cfg *GirlProjectConfig) *PreflightResult {
	profiles := uniqueSorted(cfg.Profiles)
	checks := []PreflightCheck{
		checkPackageManager(path),
		checkReadme(path),
		checkGitIgnore(path),
		checkLicense(path),
	}

	for _, profile := range profiles {
		switch profile {
		case "cai":
			checks = append(checks,
				checkCAIDirectory(path),
				checkSIGILManifest(path),
				checkPreflightConfig(path),
				checkLaunchKit(path),
				checkTenancy(path),
				checkGirlInstalled(),
				checkCAIAgents(path),
				checkAgentsMd(path),
				checkGrpDir(path),
				checkVerificationScripts(path),
				checkSecretFiles(path),
			)
		case "next":
			checks = append(checks, checkNextJSConfig(path))
		case "convex":
			checks = append(checks, checkConvexConfig(path))
		case "clerk":
			checks = append(checks, checkClerkConfig(path))
		case "vercel":
			checks = append(checks, checkVercelConfig(path))
		case "cloudflare":
			checks = append(checks, checkCloudflareProfile(path))
		case "railway":
			checks = append(checks, checkRailwayProfile(path))
		case "monorepo":
			checks = append(checks, checkMonorepoProfile(path))
		}
	}

	result := &PreflightResult{
		SpecVersion: "1.0",
		ID:          fmt.Sprintf("preflight_%d", time.Now().Unix()),
		Type:        "cai-preflight",
		Path:        path,
		Profile:     strings.Join(profiles, ","),
		Checks:      dedupePreflightChecks(checks),
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
	if result.Profile == "" {
		result.Profile = "generic"
	}
	finalizeProjectPreflight(result)
	return result
}

func checkCloudflareProfile(path string) PreflightCheck {
	if fileExistsAny(path, "wrangler.toml", "wrangler.json", "wrangler.jsonc") {
		return PreflightCheck{Name: "Cloudflare Config", Status: "pass", Message: "Cloudflare Wrangler configuration detected"}
	}
	return PreflightCheck{Name: "Cloudflare Config", Status: "warn", Message: "Cloudflare profile selected but no Wrangler configuration found"}
}

func checkRailwayProfile(path string) PreflightCheck {
	if fileExistsAny(path, "railway.toml", "railway.json") {
		return PreflightCheck{Name: "Railway Config", Status: "pass", Message: "Railway configuration detected"}
	}
	return PreflightCheck{Name: "Railway Config", Status: "warn", Message: "Railway profile selected but no Railway configuration found"}
}

func checkMonorepoProfile(path string) PreflightCheck {
	pkg, err := readProjectPackageJSON(path)
	if err == nil && len(packageWorkspaces(pkg)) > 0 {
		return PreflightCheck{Name: "Monorepo Workspaces", Status: "pass", Message: "Package workspaces detected"}
	}
	return PreflightCheck{Name: "Monorepo Workspaces", Status: "warn", Message: "Monorepo profile selected but package workspaces were not detected"}
}

func dedupePreflightChecks(checks []PreflightCheck) []PreflightCheck {
	seen := map[string]bool{}
	result := make([]PreflightCheck, 0, len(checks))
	for _, check := range checks {
		if seen[check.Name] {
			continue
		}
		seen[check.Name] = true
		result = append(result, check)
	}
	return result
}

func finalizeProjectPreflight(result *PreflightResult) {
	for _, check := range result.Checks {
		switch check.Status {
		case "pass":
			result.Summary.Pass++
		case "warn":
			result.Summary.Warn++
		case "fail":
			result.Summary.Fail++
		}
	}
	result.Summary.Total = len(result.Checks)
	switch {
	case result.Summary.Fail > 0:
		result.Status = "fail"
	case result.Summary.Warn > 0:
		result.Status = "warn"
	default:
		result.Status = "pass"
	}
}
