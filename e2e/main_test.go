//go:build e2e

package e2e_test

import (
	"log"
	"os"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

var pw *playwright.Playwright
var pwErr error = nil

func TestMain(m *testing.M) {
	// Try to start the Playwright driver. On failure pw stays nil and each
	// test will call t.Skip() via newPage(). Run 'task playwright:install'
	// to install the driver and browser binaries.
	var err error
	pw, err = playwright.Run()
	if err != nil {
		pwErr = err
	}

	code := m.Run()

	if pw != nil {
		if err := pw.Stop(); err != nil {
			log.Printf("could not stop playwright: %v", err)
		}
	}
	os.Exit(code)
}

