package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mcsd/core"
)

type fileEntry struct {
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	IsDir    bool      `json:"is_dir"`
	Modified time.Time `json:"modified"`
}

func safeJoin(base, relPath string) (string, error) {
	abs := filepath.Join(base, filepath.Clean("/"+relPath))
	if abs != base && !strings.HasPrefix(abs, base+string(filepath.Separator)) {
		return "", fmt.Errorf("path outside instance directory")
	}
	return abs, nil
}

func instanceBase(r *http.Request) (string, error) {
	id := r.PathValue("id")
	if _, err := core.LoadInstanceConfig(id); err != nil {
		return "", err
	}
	return core.InstanceDir(id), nil
}

func listFiles(w http.ResponseWriter, r *http.Request) {
	base, err := instanceBase(r)
	if err != nil {
		writeError(w, err)
		return
	}
	target, err := safeJoin(base, r.URL.Query().Get("path"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	entries, err := os.ReadDir(target)
	if err != nil {
		writeError(w, err)
		return
	}
	result := make([]fileEntry, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		result = append(result, fileEntry{
			Name:     e.Name(),
			Size:     info.Size(),
			IsDir:    e.IsDir(),
			Modified: info.ModTime().UTC(),
		})
	}
	writeJSON(w, http.StatusOK, result)
}

func readFile(w http.ResponseWriter, r *http.Request) {
	base, err := instanceBase(r)
	if err != nil {
		writeError(w, err)
		return
	}
	target, err := safeJoin(base, r.URL.Query().Get("path"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	f, err := os.Open(target)
	if err != nil {
		writeError(w, err)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	io.Copy(w, f)
}

func writeFile(w http.ResponseWriter, r *http.Request) {
	base, err := instanceBase(r)
	if err != nil {
		writeError(w, err)
		return
	}
	target, err := safeJoin(base, r.URL.Query().Get("path"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := atomicWrite(target, data); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func deleteFile(w http.ResponseWriter, r *http.Request) {
	base, err := instanceBase(r)
	if err != nil {
		writeError(w, err)
		return
	}
	target, err := safeJoin(base, r.URL.Query().Get("path"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if target == base {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot delete instance root"})
		return
	}

	if err := os.RemoveAll(target); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	base, err := instanceBase(r)
	if err != nil {
		writeError(w, err)
		return
	}
	destDir, err := safeJoin(base, r.URL.Query().Get("path"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "parse multipart: " + err.Error()})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing file field"})
		return
	}
	defer file.Close()

	destPath, err := safeJoin(base, filepath.Join(r.URL.Query().Get("path"), header.Filename))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := os.MkdirAll(destDir, 0750); err != nil {
		writeError(w, err)
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := atomicWrite(destPath, data); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"path": strings.TrimPrefix(destPath, base+string(filepath.Separator))})
}

func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0640); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
