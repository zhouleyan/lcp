package rest

import (
	"fmt"
	"net/http"
	"strings"

	"lcp.io/lcp/lib/runtime"
)

// FileResponse is a special response type for file downloads.
// When returned from a HandlerFunc, the framework writes raw bytes
// with Content-Disposition instead of JSON/YAML serialization.
type FileResponse struct {
	runtime.TypeMeta
	FileName    string
	ContentType string
	Data        []byte
}

func (f *FileResponse) GetTypeMeta() *runtime.TypeMeta { return &f.TypeMeta }

// writeFileResponse writes a FileResponse to the HTTP response writer.
func writeFileResponse(w http.ResponseWriter, statusCode int, fr *FileResponse) {
	// Sanitize filename to prevent header injection
	safeName := strings.NewReplacer(`"`, "", "\r", "", "\n", "").Replace(fr.FileName)
	w.Header().Set("Content-Type", fr.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, safeName))
	w.WriteHeader(statusCode)
	_, _ = w.Write(fr.Data)
}
