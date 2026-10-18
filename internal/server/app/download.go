package app

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// downloadDirectory is resolved against the process working directory, the same
// base the environment file already uses. Operators drop release archives here;
// the server only ever reads, so replacing a package needs no rebuild.
const downloadDirectory = "public"

// downloadContentTypes fixes the type per extension. The embedded Web assets
// keep their own map in static.go; this one covers operator-supplied packages,
// so a wrong extension is a visible download failure rather than a sniffed type.
var downloadContentTypes = map[string]string{
	".zip": "application/zip",
	".exe": "application/vnd.microsoft.portable-executable",
	".tar": "application/x-tar",
	".gz":  "application/gzip",
	".dmg": "application/x-apple-diskimage",
	".pkg": "application/octet-stream",
}

// handleDownload serves one file from the operator's download directory.
//
// Only GET and HEAD are allowed. The request path is reduced to its base name
// before it touches the filesystem, so a traversal segment cannot escape that
// directory and a hidden file cannot be addressed.
func (s *Server) handleDownload(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		writer.Header().Set("Allow", "GET, HEAD")
		sendJSON(writer, http.StatusMethodNotAllowed, errorBody{"Method not allowed"})
		return
	}
	name := path.Base(request.URL.Path)
	if name == "" || name == "." || name == "/" || strings.HasPrefix(name, ".") {
		sendJSON(writer, http.StatusNotFound, errorBody{"Not found"})
		return
	}
	file, err := os.Open(filepath.Join(downloadDirectory, name))
	if err != nil {
		sendJSON(writer, http.StatusNotFound, errorBody{"Not found"})
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		sendJSON(writer, http.StatusNotFound, errorBody{"Not found"})
		return
	}
	writer.Header().Set("Content-Type", downloadContentTypes[strings.ToLower(path.Ext(name))])
	writer.Header().Set("Content-Disposition", "attachment; filename=\""+name+"\"")
	// A package is replaced in place, so a cached copy would serve a stale build.
	writer.Header().Set("Cache-Control", "no-store")
	http.ServeContent(writer, request, name, info.ModTime(), file)
}
