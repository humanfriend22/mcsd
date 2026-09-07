package main

import (
	"fmt"
	"os/exec"
	"strings"
)


func main() {
		// 1. Look for java in PATH
		path, err := exec.LookPath("java")
		if err == nil {
			fmt.Println("Found in PATH:", path)
		}

		// 2. Execute /usr/libexec/java_home (macOS specific)
		cmd := exec.Command("/usr/libexec/java_home")

		// Output() runs the command and returns its standard output as a []byte
		stdout, err := cmd.Output()
		if err != nil {
			fmt.Println("Error running java_home:", err)
			return
		}

		// Trim whitespace/newlines from the captured output
		javaHome := strings.TrimSpace(string(stdout))
		fmt.Println("JAVA_HOME from helper:", javaHome)
}
