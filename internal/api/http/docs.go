package apihttp

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Y4shin/open-caucus/internal/docs"
)

func NewDocsAssetHandler(service *docs.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			http.NotFound(w, r)
			return
		}

		assetPath := r.PathValue("assetPath")
		if assetPath == "" {
			assetPath = strings.TrimPrefix(r.URL.Path, "/docs/assets/")
			assetPath = strings.TrimPrefix(assetPath, "docs/assets/")
		}

		body, contentType, err := service.ReadAsset(assetPath)
		if err != nil {
			if errors.Is(err, docs.ErrNotFound) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "failed to read docs asset", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "public, max-age=600")
		_, _ = w.Write(body)
	}
}
