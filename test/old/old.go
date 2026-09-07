package old

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/coreos/go-systemd/v22/dbus"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <service-name>\n", os.Args[0])
		os.Exit(1)
	}
	unit := os.Args[1]

	ctx := context.Background()
	conn, err := dbus.NewSystemdConnectionContext(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "d-bus connect: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	// Unit status (ActiveState, SubState, Description, etc.)
	statuses, err := conn.ListUnitsByNamesContext(ctx, []string{unit})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ListUnitsByNames: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("=== Unit Status ===")
	if len(statuses) > 0 {
		s := statuses[0]
		fmt.Printf("  Name:        %s\n", s.Name)
		fmt.Printf("  Description: %s\n", s.Description)
		fmt.Printf("  LoadState:   %s\n", s.LoadState)
		fmt.Printf("  ActiveState: %s\n", s.ActiveState)
		fmt.Printf("  SubState:    %s\n", s.SubState)
	} else {
		fmt.Println("  (no unit found)")
	}

	// Unit file state (enabled/disabled/static/indirect)
	// For template instances like mcsd-instance@test.service, the file on disk is mcsd-instance@.service
	patterns := []string{unit}
	if i := strings.Index(unit, "@"); i >= 0 {
		patterns = append(patterns, unit[:i+1]+".service")
	}
	files, err := conn.ListUnitFilesByPatternsContext(ctx, nil, patterns)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ListUnitFilesByPatterns: %v\n", err)
	} else {
		fmt.Println("\n=== Unit File ===")
		if len(files) > 0 {
			f := files[0]
			fmt.Printf("  Path:  %s\n", f.Path)
			fmt.Printf("  Type:  %s\n", f.Type)
		} else {
			fmt.Println("  (no unit file found)")
		}
	}

	// All properties via GetUnitProperties
	props, err := conn.GetUnitPropertiesContext(ctx, unit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetUnitProperties: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("\n=== All Properties ===")
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(props); err != nil {
		fmt.Fprintf(os.Stderr, "json encode: %v\n", err)
		os.Exit(1)
	}
}
