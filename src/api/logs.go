package api

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"mcsd/core"
)

func streamLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := core.LoadInstance(id); err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	logPath := filepath.Join(core.InstanceDir(id), "logs", "latest.log")

	if _, err := os.Stat(logPath); err != nil {
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if ok {
			fmt.Fprintf(w, "data: latest.log not found\n\n")
			flusher.Flush()
		}
		return
	}

	cmd := exec.CommandContext(r.Context(), "tail", "-n", "1000", "-F", logPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		writeError(w, err)
		return
	}
	if err := cmd.Start(); err != nil {
		writeError(w, err)
		return
	}
	defer cmd.Wait()

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, fmt.Errorf("streaming unsupported"))
		return
	}

	w.WriteHeader(http.StatusOK)

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		fmt.Fprintf(w, "data: %s\n\n", scanner.Text())
		flusher.Flush()
	}
	if err := scanner.Err(); err != nil {
		return
	}
}
