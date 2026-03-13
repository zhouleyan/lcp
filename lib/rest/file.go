package rest

import (
	"fmt"
	"net/http"

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
	w.Header().Set("Content-Type", fr.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fr.FileName))
	w.WriteHeader(statusCode)
	_, _ = w.Write(fr.Data)
}
