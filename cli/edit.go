package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"mcsd/core"
)

type EditCmd struct {
	ID string `arg:"" help:"InstanceConfig ID"`
}

func (c *EditCmd) Run() error {
	if err := core.EnsureReady(); err != nil {
		return err
	}
	instance, err := loadInstance(c.ID)
	if err != nil {
		return err
	}

	raw, err := json.Marshal(instance)
	if err != nil {
		return fmt.Errorf("marshal instance: %w", err)
	}
	var fields map[string]any
	json.Unmarshal(raw, &fields)
	delete(fields, "id")
	editable, _ := json.MarshalIndent(fields, "", "  ")

	tmp, err := os.CreateTemp("", "mcsd-edit-*.json")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(editable); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	tmp.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano"
	}
	editorCmd := exec.Command(editor, tmpPath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr
	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("editor: %w", err)
	}

	updated, err := os.ReadFile(tmpPath)
	if err != nil {
		return fmt.Errorf("read edited file: %w", err)
	}

	var edited core.InstanceConfig
	if err := json.Unmarshal(updated, &edited); err != nil {
		return fmt.Errorf("parse edited JSON: %w", err)
	}
	edited.ID = c.ID

	if err := edited.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := core.WriteInstanceConfig(c.ID, &edited); err != nil {
		return fmt.Errorf("save instance: %w", err)
	}

	fmt.Printf("Updated %s.\n", c.ID)
	return nil
}
