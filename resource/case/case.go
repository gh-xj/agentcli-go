package caseresource

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	agentcli "github.com/gh-xj/agentcli-go"
	"github.com/gh-xj/agentcli-go/dal"
	"github.com/gh-xj/agentcli-go/resource"
	"github.com/gh-xj/agentcli-go/strategy"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// CaseResource implements the Resource, Validator, and Transitioner interfaces.
type CaseResource struct {
	fs    dal.FileSystem
	exec  dal.Executor
	strat *strategy.Strategy
	sm    *StateMachine
}

// Compile-time interface checks.
var (
	_ resource.Resource     = (*CaseResource)(nil)
	_ resource.Validator    = (*CaseResource)(nil)
	_ resource.Transitioner = (*CaseResource)(nil)
)

// New creates a new CaseResource.
func New(fs dal.FileSystem, exec dal.Executor, strat *strategy.Strategy) *CaseResource {
	cr := &CaseResource{
		fs:    fs,
		exec:  exec,
		strat: strat,
	}
	if strat != nil {
		cr.sm = NewStateMachine(strat.Transitions)
	}
	return cr
}

// casesDir resolves the cases directory based on storage backend configuration.
func (cr *CaseResource) casesDir() (string, error) {
	if cr.strat == nil {
		return "", fmt.Errorf("no strategy loaded")
	}

	switch cr.strat.Storage.Backend {
	case "in-repo":
		return filepath.Join(cr.strat.Root, "cases"), nil
	default:
		// separate-repo (default)
		caseRepoPath := cr.strat.Storage.CaseRepoPath
		if caseRepoPath == "" {
			base := filepath.Base(cr.strat.Root)
			caseRepoPath = filepath.Join("..", base+"-cases")
		}
		// Resolve relative to the project root.
		if !filepath.IsAbs(caseRepoPath) {
			caseRepoPath = filepath.Join(cr.strat.Root, caseRepoPath)
		}
		return filepath.Join(caseRepoPath, "cases"), nil
	}
}

// Schema returns the resource schema for cases.
func (cr *CaseResource) Schema() resource.ResourceSchema {
	var statuses []string
	if cr.sm != nil {
		statuses = cr.sm.AllStatuses()
	}

	return resource.ResourceSchema{
		Kind: "case",
		Fields: []resource.FieldDef{
			{Name: "id", Type: "string", Required: true},
			{Name: "type", Type: "string", Required: true},
			{Name: "status", Type: "string", Required: true},
			{Name: "claimed_by", Type: "string", Required: false},
			{Name: "created", Type: "string", Required: true},
		},
		Statuses: statuses,
		CreateArgs: []resource.ArgDef{
			{Name: "slug", Description: "URL-safe case identifier", Required: true},
		},
		Description: "A case record tracking an operational task through its lifecycle.",
	}
}

// Create creates a new case directory and case.md file.
func (cr *CaseResource) Create(ctx *agentcli.AppContext, slug string, opts map[string]string) (*resource.Record, error) {
	if cr.strat == nil {
		return nil, fmt.Errorf("no strategy loaded")
	}

	if err := validateSlug(slug); err != nil {
		return nil, err
	}

	casesRoot, err := cr.casesDir()
	if err != nil {
		return nil, err
	}

	if err := cr.fs.EnsureDir(casesRoot); err != nil {
		return nil, fmt.Errorf("ensure cases dir: %w", err)
	}

	dateStr := time.Now().Format("20060102")
	baseName := fmt.Sprintf("CASE-%s-%s", dateStr, slug)
	dirName := baseName
	caseDir := filepath.Join(casesRoot, dirName)

	// Handle collision with -02, -03 suffix.
	suffix := 2
	for cr.fs.Exists(caseDir) {
		dirName = fmt.Sprintf("%s-%02d", baseName, suffix)
		caseDir = filepath.Join(casesRoot, dirName)
		suffix++
	}

	if err := cr.fs.EnsureDir(caseDir); err != nil {
		return nil, fmt.Errorf("create case dir: %w", err)
	}

	// Build case.md content from strategy's SchemaTemplate if available.
	fm := Frontmatter{
		Type:      "intake",
		Status:    cr.sm.Initial(),
		ClaimedBy: "none",
		Created:   dateStr,
	}

	var content string
	if cr.strat.SchemaTemplate != "" {
		// Parse template frontmatter and override with runtime values.
		tplFM, body, err := ParseFrontmatter(cr.strat.SchemaTemplate)
		if err == nil {
			// Use template values as defaults, override with runtime.
			if tplFM.Type != "" {
				fm.Type = tplFM.Type
			}
			fm.Status = cr.sm.Initial()
			if tplFM.ClaimedBy != "" && tplFM.ClaimedBy != "none" {
				fm.ClaimedBy = tplFM.ClaimedBy
			}
			fm.Created = dateStr

			// Replace title placeholder.
			body = strings.Replace(body, "# Case Title", "# "+dirName, 1)
			content = RenderFrontmatter(fm) + body
		} else {
			content = RenderFrontmatter(fm) + "# " + dirName + "\n"
		}
	} else {
		content = RenderFrontmatter(fm) + "# " + dirName + "\n"
	}

	caseMDPath := filepath.Join(caseDir, "case.md")
	if err := cr.fs.WriteFile(caseMDPath, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("write case.md: %w", err)
	}

	return cr.recordFromFrontmatter(dirName, caseMDPath, fm), nil
}

// List walks the cases directory and returns matching records.
func (cr *CaseResource) List(ctx *agentcli.AppContext, filter resource.Filter) ([]resource.Record, error) {
	if cr.strat == nil {
		return nil, fmt.Errorf("no strategy loaded")
	}

	casesRoot, err := cr.casesDir()
	if err != nil {
		return nil, err
	}

	entries, err := cr.fs.ReadDir(casesRoot)
	if err != nil {
		// If directory doesn't exist, return empty list.
		return nil, nil
	}

	// Pre-compute status filter if specified.
	var statusFilter map[string]bool
	if statusVal, ok := filter["status"]; ok && statusVal != "" {
		statusFilter, err = cr.sm.ExpandStatusFilter(statusVal)
		if err != nil {
			return nil, err
		}
	}

	slotFilter := ""
	if slot, ok := filter["slot"]; ok {
		slotFilter = slot
	}

	var records []resource.Record
	for _, entry := range entries {
		if !entry.IsDir || !strings.HasPrefix(entry.Name, "CASE-") {
			continue
		}

		caseMDPath := filepath.Join(casesRoot, entry.Name, "case.md")
		data, err := cr.fs.ReadFile(caseMDPath)
		if err != nil {
			continue
		}

		fm, _, err := ParseFrontmatter(string(data))
		if err != nil {
			continue
		}

		// Apply filters.
		if statusFilter != nil && !statusFilter[fm.Status] {
			continue
		}
		if slotFilter != "" && fm.ClaimedBy != slotFilter {
			continue
		}

		records = append(records, *cr.recordFromFrontmatter(entry.Name, caseMDPath, fm))
	}

	return records, nil
}

// Get retrieves a case record by its ID.
func (cr *CaseResource) Get(ctx *agentcli.AppContext, id string) (*resource.Record, error) {
	if cr.strat == nil {
		return nil, fmt.Errorf("no strategy loaded")
	}

	caseMDPath, err := cr.findCaseMD(id)
	if err != nil {
		return nil, err
	}

	data, err := cr.fs.ReadFile(caseMDPath)
	if err != nil {
		return nil, fmt.Errorf("read case.md: %w", err)
	}

	fm, _, err := ParseFrontmatter(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}

	return cr.recordFromFrontmatter(id, caseMDPath, fm), nil
}

// Validate checks that a case has all required frontmatter fields.
func (cr *CaseResource) Validate(ctx *agentcli.AppContext, id string) (*agentcli.DoctorReport, error) {
	if cr.strat == nil {
		return nil, fmt.Errorf("no strategy loaded")
	}

	caseMDPath, err := cr.findCaseMD(id)
	if err != nil {
		return nil, err
	}

	data, err := cr.fs.ReadFile(caseMDPath)
	if err != nil {
		return nil, fmt.Errorf("read case.md: %w", err)
	}

	report := &agentcli.DoctorReport{
		SchemaVersion: "1.0",
		OK:            true,
	}

	fm, _, fmErr := ParseFrontmatter(string(data))
	if fmErr != nil {
		report.OK = false
		report.Findings = append(report.Findings, agentcli.DoctorFinding{
			Code:    "missing_frontmatter",
			Path:    caseMDPath,
			Message: "case.md has no valid YAML frontmatter",
		})
		return report, nil
	}

	if fm.Type == "" {
		report.OK = false
		report.Findings = append(report.Findings, agentcli.DoctorFinding{
			Code:    "missing_field",
			Path:    caseMDPath,
			Message: "missing required field: type",
		})
	}
	if fm.Status == "" {
		report.OK = false
		report.Findings = append(report.Findings, agentcli.DoctorFinding{
			Code:    "missing_field",
			Path:    caseMDPath,
			Message: "missing required field: status",
		})
	}
	if fm.Created == "" {
		report.OK = false
		report.Findings = append(report.Findings, agentcli.DoctorFinding{
			Code:    "missing_field",
			Path:    caseMDPath,
			Message: "missing required field: created",
		})
	}

	return report, nil
}

// Transition applies a state machine action to a case and returns the updated record.
func (cr *CaseResource) Transition(ctx *agentcli.AppContext, id string, action string) (*resource.Record, error) {
	if cr.strat == nil {
		return nil, fmt.Errorf("no strategy loaded")
	}

	caseMDPath, err := cr.findCaseMD(id)
	if err != nil {
		return nil, err
	}

	data, err := cr.fs.ReadFile(caseMDPath)
	if err != nil {
		return nil, fmt.Errorf("read case.md: %w", err)
	}

	fm, body, err := ParseFrontmatter(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}

	oldCategory := cr.sm.CategoryForStatus(fm.Status)

	newStatus, err := cr.sm.Apply(fm.Status, action)
	if err != nil {
		return nil, err
	}

	fm.Status = newStatus
	newContent := RenderFrontmatter(fm) + body

	if err := cr.fs.WriteFile(caseMDPath, []byte(newContent), 0o644); err != nil {
		return nil, fmt.Errorf("write case.md: %w", err)
	}

	// Check if category changed — for now this is informational.
	// In separate-repo mode with group subdirs, we would move the directory.
	// In in-repo mode, cases stay in the same flat directory.
	newCategory := cr.sm.CategoryForStatus(newStatus)
	_ = oldCategory
	_ = newCategory

	return cr.recordFromFrontmatter(id, caseMDPath, fm), nil
}

// findCaseMD locates the case.md file for a given case ID.
func (cr *CaseResource) findCaseMD(id string) (string, error) {
	casesRoot, err := cr.casesDir()
	if err != nil {
		return "", err
	}

	// Direct lookup in the flat cases directory.
	caseMDPath := filepath.Join(casesRoot, id, "case.md")
	if cr.fs.Exists(caseMDPath) {
		return caseMDPath, nil
	}

	return "", fmt.Errorf("case %q not found", id)
}

// recordFromFrontmatter builds a Record from a case ID and its frontmatter.
func (cr *CaseResource) recordFromFrontmatter(id, rawPath string, fm Frontmatter) *resource.Record {
	return &resource.Record{
		Kind: "case",
		ID:   id,
		Fields: map[string]any{
			"id":         id,
			"type":       fm.Type,
			"status":     fm.Status,
			"claimed_by": fm.ClaimedBy,
			"created":    fm.Created,
		},
		RawPath: rawPath,
	}
}

// validateSlug checks that a case slug is safe and well-formed.
func validateSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("slug cannot be empty")
	}
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("invalid slug %q: must match ^[a-z0-9][a-z0-9-]*$", slug)
	}
	if len(slug) > 128 {
		return fmt.Errorf("invalid slug %q: max 128 characters", slug)
	}
	return nil
}
