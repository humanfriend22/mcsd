package api

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mcsd/core"
	"mcsd/utils"
)

type fileEntry struct {
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	IsDir    bool      `json:"is_dir"`
	Modified time.Time `json:"modified"`
}

// extractTargetPath resolves the instance base dir and jails the request's
// ?path= query param under it. Returns base too since deleteFile and
// uploadFile need it for extra checks beyond the target path itself.
func extractTargetPath(r *http.Request) (basePath string, targetPath string, err error) {
	// Instance Directory
	id := r.PathValue("id")
	instance, loadErr := core.LoadInstance(id)
	if loadErr != nil {
		return "", "", loadErr
	}
	basePath = core.InstanceDir(instance.ID)

	// Target Path
	targetPath, err = utils.SafeJoin(basePath, r.URL.Query().Get("path"))
	if err != nil {
		return "", "", err
	}
	return basePath, targetPath, nil
}

func listFiles(w http.ResponseWriter, r *http.Request) {
	_, target, err := extractTargetPath(r)
	if err != nil {
		writeError(w, err)
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
	_, target, err := extractTargetPath(r)
	if err != nil {
		writeError(w, err)
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
	_, target, err := extractTargetPath(r)
	if err != nil {
		writeError(w, err)
		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := utils.WriteFile(target, data, 0640); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func deleteFile(w http.ResponseWriter, r *http.Request) {
	base, target, err := extractTargetPath(r)
	if err != nil {
		writeError(w, err)
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
	base, destDir, err := extractTargetPath(r)
	if err != nil {
		writeError(w, err)
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

	destPath, err := utils.SafeJoin(base, filepath.Join(r.URL.Query().Get("path"), header.Filename))
	if err != nil {
		writeError(w, err)
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
	if err := utils.WriteFile(destPath, data, 0640); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"path": strings.TrimPrefix(destPath, base+string(filepath.Separator))})
}
