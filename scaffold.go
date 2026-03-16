package agentcli

import "encoding/json"

// DoctorFinding describes a single compliance issue in a scaffolded project.
type DoctorFinding struct {
	Code    string `json:"code"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

// DoctorReport summarizes scaffold compliance checks.
type DoctorReport struct {
	SchemaVersion string          `json:"schema_version"`
	OK            bool            `json:"ok"`
	Findings      []DoctorFinding `json:"findings"`
}

// JSON returns the report as indented JSON.
func (r DoctorReport) JSON() (string, error) {
	out, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}
