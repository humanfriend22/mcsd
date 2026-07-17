package api

import (
	"bufio"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"mcsd/core"
)

func streamLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := cachedInstanceOrError(id); err != nil {
		writeError(w, err)
		return
	}

	unit := core.UnitName(id)
	since, _ := sdClient.ActiveSince(unit)

	args := []string{
		"-u", unit,
		"--output=cat",
		"--no-pager",
		"-f",
	}
	if !since.IsZero() {
		args = append(args, fmt.Sprintf("--since=%s", since.Format(time.DateTime)))
	}

	cmd := exec.CommandContext(r.Context(), "journalctl", args...)
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

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, fmt.Errorf("streaming unsupported"))
		return
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		fmt.Fprintf(w, "data: %s\n\n", scanner.Text())
		flusher.Flush()
	}
	if err := scanner.Err(); err != nil {
		writeError(w, err)
		return
	}
}
