package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
)

type schema struct {
	SchemaVersion string   `json:"schema_version"`
	RequiredKeys  []string `json:"required_keys"`
}

func main() {
	schemaPath := flag.String("schema", "", "path to schema file")
	inputPath := flag.String("input", "", "path to smoke output json")
	flag.Parse()

	if *schemaPath == "" || *inputPath == "" {
		fmt.Fprintln(os.Stderr, "usage: smokecheck --schema <schema.json> --input <output.json>")
		os.Exit(2)
	}
	if err := run(*schemaPath, *inputPath); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	fmt.Println("smoke schema check passed")
}

func run(schemaPath, inputPath string) error {
	s, err := readSchema(schemaPath)
	if err != nil {
		return err
	}
	payload, err := readPayload(inputPath)
	if err != nil {
		return err
	}
	for _, key := range s.RequiredKeys {
		if _, ok := payload[key]; !ok {
			return fmt.Errorf("missing required key: %s", key)
		}
	}
	if got, _ := payload["schema_version"].(string); got != s.SchemaVersion {
		return fmt.Errorf("schema_version mismatch: got %q want %q", got, s.SchemaVersion)
	}
	return nil
}

func readSchema(path string) (schema, error) {
	var out schema
	data, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, err
	}
	if out.SchemaVersion == "" {
		return out, errors.New("schema_version is required in schema")
	}
	if len(out.RequiredKeys) == 0 {
		return out, errors.New("required_keys must not be empty")
	}
	return out, nil
}

func readPayload(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
