package core

import (
	"errors"
	. "mcsd/utils"
	"path/filepath"
	"strings"
)

type Properties map[string]string

func (instance *Instance) ReadProperties() (Properties, error) {
	data, err := ReadFile(filepath.Join(InstanceDir(instance.ID), "server.properties"))
	if err != nil {
		return nil, err
	}

	props := make(Properties)
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") { continue }
		key, value, ok := strings.Cut(line, "=")
		if ok {
			props[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return props, nil
}

func (instance *Instance) WriteProperties(props Properties, overwrite bool) error {
	out := props
	if !overwrite {
		existing, err := instance.ReadProperties()
		if err != nil {
			if _, ok := errors.AsType[*NotFoundError](err); !ok {
				return err
			}
			existing = make(Properties)
		}
		for key, value := range props {
			existing[key] = value
		}
		out = existing
	}

	var b strings.Builder
	for key, value := range out {
		b.WriteString(key)
		b.WriteString("=")
		b.WriteString(value)
		b.WriteString("\n")
	}
	return WriteFile(filepath.Join(InstanceDir(instance.ID), "server.properties"), []byte(b.String()), 0640)
}
