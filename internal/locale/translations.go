package locale

import (
	"embed"
	"io/fs"
	"sync"

	"github.com/invopop/ctxi18n"
)

//go:embed locales/*.yaml
var translationsFS embed.FS
var (
	loadTranslationsOnce sync.Once
	loadTranslationsErr  error
)

func LoadTranslations() error {
	loadTranslationsOnce.Do(func() {
		var root fs.FS = translationsFS
		if sub, err := fs.Sub(translationsFS, "locales"); err == nil {
			root = sub
		}
		loadTranslationsErr = ctxi18n.Load(root)
	})
	return loadTranslationsErr
}
