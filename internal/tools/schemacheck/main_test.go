package main

import (
	"path/filepath"
	"testing"
)

func TestRun(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	tests := []struct {
		name       string
		schemaPath string
		inputPath  string
		wantErr    bool
	}{
		{
			name:       "doctor ok",
			schemaPath: filepath.Join(root, "schemas", "doctor-report.schema.json"),
			inputPath:  filepath.Join(root, "testdata", "contracts", "doctor-report.ok.json"),
			wantErr:    false,
		},
		{
			name:       "doctor missing findings",
			schemaPath: filepath.Join(root, "schemas", "doctor-report.schema.json"),
			inputPath:  filepath.Join(root, "testdata", "contracts", "doctor-report.bad-missing-findings.json"),
			wantErr:    true,
		},
		{
			name:       "doctor bad schema version",
			schemaPath: filepath.Join(root, "schemas", "doctor-report.schema.json"),
			inputPath:  filepath.Join(root, "testdata", "contracts", "doctor-report.bad-schema-version.json"),
			wantErr:    true,
		},
		{
			name:       "version ok",
			schemaPath: filepath.Join(root, "schemas", "scaffold-version-output.schema.json"),
			inputPath:  filepath.Join(root, "testdata", "contracts", "scaffold-version-output.ok.json"),
			wantErr:    false,
		},
		{
			name:       "version missing date",
			schemaPath: filepath.Join(root, "schemas", "scaffold-version-output.schema.json"),
			inputPath:  filepath.Join(root, "testdata", "contracts", "scaffold-version-output.bad-missing-date.json"),
			wantErr:    true,
		},
		{
			name:       "version bad schema version",
			schemaPath: filepath.Join(root, "schemas", "scaffold-version-output.schema.json"),
			inputPath:  filepath.Join(root, "testdata", "contracts", "scaffold-version-output.bad-schema-version.json"),
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := run(tc.schemaPath, tc.inputPath)
			if tc.wantErr && err == nil {
				t.Fatal("expected error but got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
