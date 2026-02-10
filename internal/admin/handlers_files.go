package admin

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"mime/multipart"
)

const (
	stagingDir = "_uploads"
	publicDir  = "assets/common_files"
)

type UploadedFile struct {
	Name string
	Size string
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	// -------- GET: list + form --------
	case http.MethodGet:
		files, err := listPublishedFiles()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tmpl, err := template.ParseFiles(
			"internal/templates/base.html",
			"internal/templates/admin/files.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tmpl.ExecuteTemplate(w, "base", struct {
			Files []UploadedFile
		}{
			Files: files,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	// -------- POST: upload (atomic publish) --------
	case http.MethodPost:
		if err := r.ParseMultipartForm(64 << 20); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		files := r.MultipartForm.File["files"]
		if len(files) == 0 {
			http.Error(w, "no files uploaded", http.StatusBadRequest)
			return
		}

		if err := os.MkdirAll(publicDir, 0o755); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// publish files ONE BY ONE, atomically
		for _, fh := range files {
			if err := publishFileAtomically(fh); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		http.Redirect(w, r, "/admin/files", http.StatusSeeOther)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func publishFileAtomically(fh *multipart.FileHeader) error {
	// 1. Sanitize filename (security boundary)
	name := filepath.Base(fh.Filename)
	if name == "" || name == "." || name == ".." {
		return fmt.Errorf("invalid filename")
	}

	// 2. Paths
	tmpPath := filepath.Join(stagingDir, "."+name+".tmp")
	finalPath := filepath.Join(publicDir, name)

	// 3. Open uploaded file
	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	// 4. Create temp file in staging dir
	tmp, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	// 5. Write full contents
	if _, err := io.Copy(tmp, src); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}

	// 6. Flush file contents to disk
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}

	// 7. Close before rename
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	// 8. 🔑 ATOMIC PUBLISH
	if err := os.Rename(tmpPath, finalPath); err != nil {
		os.Remove(tmpPath)
		return err
	}

	// (Optional but excellent) fsync the directory entry
	if dir, err := os.Open(publicDir); err == nil {
		dir.Sync()
		dir.Close()
	}

	return nil
}

func listPublishedFiles() ([]UploadedFile, error) {
	entries, err := os.ReadDir(publicDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var files []UploadedFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		info, err := e.Info()
		if err != nil {
			continue
		}

		name := info.Name()

		// Skip hidden / temp files
		if name == "" || name[0] == '.' {
			continue
		}

		files = append(files, UploadedFile{
			Name: name,
			Size: humanSize(info.Size()),
		})
	}

	return files, nil
}

func (s *Server) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := filepath.Base(
		r.URL.Path[len("/admin/files/delete/"):],
	)
	if name == "" {
		http.NotFound(w, r)
		return
	}

	path := filepath.Join(publicDir, name)

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/files", http.StatusSeeOther)
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for n/div >= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(n)/float64(div),
		"KMGTPE"[exp],
	)
}
