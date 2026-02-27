package docscapture

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	playwright "github.com/playwright-community/playwright-go"
)

const (
	defaultBaseURL        = "http://127.0.0.1:8080"
	defaultOutDir         = "docs/assets/captures"
	defaultViewportWidth  = 1440
	defaultViewportHeight = 900
	defaultScreenshotName = "example-admin-login.png"
	defaultGIFName        = "example-login-flow.gif"
	defaultGIFFPS         = 8
	defaultGIFScaleWidth  = 960
	defaultGIFMaxColors   = 56
	defaultTypingDelayMS  = 85.0
	defaultFinalPauseMS   = 800.0
	defaultGIFLossy       = 180
)

type CommonOptions struct {
	BaseURL        string
	OutDir         string
	ViewportWidth  int
	ViewportHeight int
	Headed         bool
	SlowMoMS       float64
}

type Theme string

const (
	ThemeLight Theme = "light"
	ThemeDark  Theme = "dark"
)

type Language string

const (
	LanguageEnglish Language = "en"
	LanguageGerman  Language = "de"
)

type Variant struct {
	Theme    Theme
	Language Language
}

type ScreenshotExampleOptions struct {
	Common     CommonOptions
	Variant    Variant
	Path       string
	OutputName string
	FullPage   bool
}

type GIFExampleOptions struct {
	Common        CommonOptions
	Variant       Variant
	StartPath     string
	LoginPath     string
	OutputName    string
	FPS           int
	ScaleWidth    int
	TypingDelayMS float64
	FinalPauseMS  float64
}

type GIFExampleResult struct {
	GIFPath string
}

func RunScreenshotExample(opts ScreenshotExampleOptions) (string, error) {
	opts = withScreenshotDefaults(opts)
	var err error
	opts.Variant, err = normalizeVariant(opts.Variant)
	if err != nil {
		return "", fmt.Errorf("normalize capture variant: %w", err)
	}

	targetURL, err := resolveURL(opts.Common.BaseURL, opts.Path)
	if err != nil {
		return "", err
	}

	outputPath, err := buildOutputPath(opts.Common.OutDir, opts.OutputName)
	if err != nil {
		return "", err
	}

	browser, cleanup, err := newBrowser(opts.Common)
	if err != nil {
		return "", err
	}
	defer cleanup()

	ctx, err := newContext(browser, opts.Common, opts.Variant, "")
	if err != nil {
		return "", err
	}
	defer ctx.Close()

	page, err := ctx.NewPage()
	if err != nil {
		return "", fmt.Errorf("create page: %w", err)
	}

	if _, err := page.Goto(targetURL, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateLoad}); err != nil {
		return "", fmt.Errorf("navigate to %q: %w", targetURL, err)
	}
	if err := page.Locator("body").WaitFor(); err != nil {
		return "", fmt.Errorf("wait for body on %q: %w", targetURL, err)
	}

	if _, err := page.Screenshot(playwright.PageScreenshotOptions{
		Path:       playwright.String(outputPath),
		FullPage:   playwright.Bool(opts.FullPage),
		Type:       playwright.ScreenshotTypePng,
		Animations: playwright.ScreenshotAnimationsDisabled,
		Caret:      playwright.ScreenshotCaretHide,
		Scale:      playwright.ScreenshotScaleCss,
	}); err != nil {
		return "", fmt.Errorf("capture screenshot: %w", err)
	}

	return outputPath, nil
}

func RunGIFExample(opts GIFExampleOptions) (GIFExampleResult, error) {
	opts = withGIFDefaults(opts)
	var err error
	opts.Variant, err = normalizeVariant(opts.Variant)
	if err != nil {
		return GIFExampleResult{}, fmt.Errorf("normalize capture variant: %w", err)
	}

	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return GIFExampleResult{}, fmt.Errorf("ffmpeg not found in PATH; install ffmpeg for GIF generation")
	}
	gifsiclePath, _ := exec.LookPath("gifsicle")

	startURL, err := resolveURL(opts.Common.BaseURL, opts.StartPath)
	if err != nil {
		return GIFExampleResult{}, err
	}
	loginURL, err := resolveURL(opts.Common.BaseURL, opts.LoginPath)
	if err != nil {
		return GIFExampleResult{}, err
	}

	gifPath, err := buildOutputPath(opts.Common.OutDir, opts.OutputName)
	if err != nil {
		return GIFExampleResult{}, err
	}

	browser, cleanup, err := newBrowser(opts.Common)
	if err != nil {
		return GIFExampleResult{}, err
	}
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "conference-tool-docs-gif-*")
	if err != nil {
		return GIFExampleResult{}, fmt.Errorf("create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	tempVideoPath := filepath.Join(tempDir, "capture.webm")

	ctx, err := newContext(browser, opts.Common, opts.Variant, tempDir)
	if err != nil {
		return GIFExampleResult{}, err
	}
	contextClosed := false
	defer func() {
		if !contextClosed {
			_ = ctx.Close()
		}
	}()

	page, err := ctx.NewPage()
	if err != nil {
		return GIFExampleResult{}, fmt.Errorf("create page: %w", err)
	}
	video := page.Video()
	if video == nil {
		return GIFExampleResult{}, fmt.Errorf("playwright video recording unavailable")
	}

	if _, err := page.Goto(startURL, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateLoad}); err != nil {
		return GIFExampleResult{}, fmt.Errorf("navigate to %q: %w", startURL, err)
	}
	if err := page.Locator("body").WaitFor(); err != nil {
		return GIFExampleResult{}, fmt.Errorf("wait for body on %q: %w", startURL, err)
	}

	if _, err := page.Goto(loginURL, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateLoad}); err != nil {
		return GIFExampleResult{}, fmt.Errorf("navigate to %q: %w", loginURL, err)
	}
	username := page.Locator("input[name=username]")
	password := page.Locator("input[name=password]")
	if err := username.WaitFor(); err != nil {
		return GIFExampleResult{}, fmt.Errorf("wait for username input: %w", err)
	}
	if err := password.WaitFor(); err != nil {
		return GIFExampleResult{}, fmt.Errorf("wait for password input: %w", err)
	}
	if err := username.Click(); err != nil {
		return GIFExampleResult{}, fmt.Errorf("focus username input: %w", err)
	}
	if err := username.PressSequentially("demo-user", playwright.LocatorPressSequentiallyOptions{
		Delay: playwright.Float(opts.TypingDelayMS),
	}); err != nil {
		return GIFExampleResult{}, fmt.Errorf("type username: %w", err)
	}
	if err := password.Click(); err != nil {
		return GIFExampleResult{}, fmt.Errorf("focus password input: %w", err)
	}
	if err := password.PressSequentially("demo-password", playwright.LocatorPressSequentiallyOptions{
		Delay: playwright.Float(opts.TypingDelayMS),
	}); err != nil {
		return GIFExampleResult{}, fmt.Errorf("type password: %w", err)
	}

	page.WaitForTimeout(opts.FinalPauseMS)

	if err := ctx.Close(); err != nil {
		return GIFExampleResult{}, fmt.Errorf("close browser context: %w", err)
	}
	contextClosed = true

	if err := video.SaveAs(tempVideoPath); err != nil {
		return GIFExampleResult{}, fmt.Errorf("save intermediate webm: %w", err)
	}

	if err := convertWebMToGIF(ffmpegPath, tempVideoPath, gifPath, opts.FPS, opts.ScaleWidth, defaultGIFMaxColors); err != nil {
		return GIFExampleResult{}, err
	}
	if gifsiclePath != "" {
		if err := optimizeGIFWithGifsicle(gifsiclePath, gifPath, defaultGIFMaxColors, defaultGIFLossy); err != nil {
			return GIFExampleResult{}, err
		}
	}

	return GIFExampleResult{GIFPath: gifPath}, nil
}

func withScreenshotDefaults(opts ScreenshotExampleOptions) ScreenshotExampleOptions {
	opts.Common = withCommonDefaults(opts.Common)
	if opts.Path == "" {
		opts.Path = "/admin/login"
	}
	if opts.OutputName == "" {
		opts.OutputName = defaultScreenshotName
	}
	return opts
}

func withGIFDefaults(opts GIFExampleOptions) GIFExampleOptions {
	opts.Common = withCommonDefaults(opts.Common)
	if opts.StartPath == "" {
		opts.StartPath = "/"
	}
	if opts.LoginPath == "" {
		opts.LoginPath = "/admin/login"
	}
	if opts.OutputName == "" {
		opts.OutputName = defaultGIFName
	}
	if opts.FPS <= 0 {
		opts.FPS = defaultGIFFPS
	}
	if opts.ScaleWidth <= 0 {
		opts.ScaleWidth = defaultGIFScaleWidth
	}
	if opts.TypingDelayMS <= 0 {
		opts.TypingDelayMS = defaultTypingDelayMS
	}
	if opts.FinalPauseMS <= 0 {
		opts.FinalPauseMS = defaultFinalPauseMS
	}
	return opts
}

func withCommonDefaults(opts CommonOptions) CommonOptions {
	if opts.BaseURL == "" {
		opts.BaseURL = defaultBaseURL
	}
	if opts.OutDir == "" {
		opts.OutDir = defaultOutDir
	}
	if opts.ViewportWidth <= 0 {
		opts.ViewportWidth = defaultViewportWidth
	}
	if opts.ViewportHeight <= 0 {
		opts.ViewportHeight = defaultViewportHeight
	}
	return opts
}

func ParseTheme(raw string) (Theme, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case "light", "lightmode":
		return ThemeLight, nil
	case "dark", "darkmode":
		return ThemeDark, nil
	default:
		return "", fmt.Errorf("unsupported theme %q (supported: light, dark)", raw)
	}
}

func ParseLanguage(raw string) (Language, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case "en", "english":
		return LanguageEnglish, nil
	case "de", "german", "deutsch":
		return LanguageGerman, nil
	default:
		return "", fmt.Errorf("unsupported language %q (supported: en/english, de/german)", raw)
	}
}

func normalizeVariant(v Variant) (Variant, error) {
	if v.Theme == "" {
		v.Theme = ThemeLight
	}
	if v.Language == "" {
		v.Language = LanguageEnglish
	}
	theme, err := ParseTheme(string(v.Theme))
	if err != nil {
		return Variant{}, err
	}
	lang, err := ParseLanguage(string(v.Language))
	if err != nil {
		return Variant{}, err
	}
	return Variant{
		Theme:    theme,
		Language: lang,
	}, nil
}

func languageLocale(language Language) string {
	switch language {
	case LanguageGerman:
		return "de-DE"
	default:
		return "en-US"
	}
}

func languageHeader(language Language) string {
	switch language {
	case LanguageGerman:
		return "de-DE,de;q=0.9,en;q=0.7"
	default:
		return "en-US,en;q=0.9,de;q=0.5"
	}
}

func newBrowser(opts CommonOptions) (playwright.Browser, func(), error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, func() {}, fmt.Errorf("start playwright: %w", err)
	}

	launchOpts := playwright.BrowserTypeLaunchOptions{}
	if opts.Headed {
		launchOpts.Headless = playwright.Bool(false)
	}
	if opts.SlowMoMS > 0 {
		launchOpts.SlowMo = playwright.Float(opts.SlowMoMS)
	}

	browser, err := pw.Chromium.Launch(launchOpts)
	if err != nil {
		_ = pw.Stop()
		return nil, func() {}, fmt.Errorf("launch chromium: %w", err)
	}

	cleanup := func() {
		_ = browser.Close()
		_ = pw.Stop()
	}
	return browser, cleanup, nil
}

func newContext(browser playwright.Browser, common CommonOptions, variant Variant, recordVideoDir string) (playwright.BrowserContext, error) {
	colorScheme := playwright.ColorSchemeLight
	if variant.Theme == ThemeDark {
		colorScheme = playwright.ColorSchemeDark
	}

	ctxOpts := playwright.BrowserNewContextOptions{
		Viewport: &playwright.Size{
			Width:  common.ViewportWidth,
			Height: common.ViewportHeight,
		},
		ColorScheme: colorScheme,
		Locale:      playwright.String(languageLocale(variant.Language)),
		ExtraHttpHeaders: map[string]string{
			"Accept-Language": languageHeader(variant.Language),
		},
	}
	if recordVideoDir != "" {
		ctxOpts.RecordVideo = &playwright.RecordVideo{
			Dir: recordVideoDir,
			Size: &playwright.Size{
				Width:  common.ViewportWidth,
				Height: common.ViewportHeight,
			},
		}
	}

	ctx, err := browser.NewContext(ctxOpts)
	if err != nil {
		return nil, fmt.Errorf("create browser context: %w", err)
	}

	baseCookieURL, err := resolveURL(common.BaseURL, "/")
	if err != nil {
		_ = ctx.Close()
		return nil, fmt.Errorf("resolve locale cookie URL: %w", err)
	}
	if err := ctx.AddCookies([]playwright.OptionalCookie{
		{
			Name:  "locale",
			Value: string(variant.Language),
			URL:   playwright.String(baseCookieURL),
		},
	}); err != nil {
		_ = ctx.Close()
		return nil, fmt.Errorf("set locale cookie: %w", err)
	}

	return ctx, nil
}

func buildOutputPath(baseDir, fileName string) (string, error) {
	if fileName == "" {
		return "", fmt.Errorf("output file name must not be empty")
	}

	absoluteBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("resolve output directory %q: %w", baseDir, err)
	}
	targetPath := filepath.Join(absoluteBase, filepath.Clean(fileName))
	relativePath, err := filepath.Rel(absoluteBase, targetPath)
	if err != nil {
		return "", fmt.Errorf("resolve output path %q: %w", fileName, err)
	}
	if relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("output file %q escapes output directory %q", fileName, absoluteBase)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return "", fmt.Errorf("create output directory for %q: %w", targetPath, err)
	}

	return targetPath, nil
}

func resolveURL(baseURL, pathOrURL string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse base URL %q: %w", baseURL, err)
	}
	if !base.IsAbs() {
		return "", fmt.Errorf("base URL must be absolute, got %q", baseURL)
	}

	if pathOrURL == "" {
		pathOrURL = "/"
	}
	target, err := url.Parse(pathOrURL)
	if err != nil {
		return "", fmt.Errorf("parse target path/URL %q: %w", pathOrURL, err)
	}
	if target.IsAbs() {
		return target.String(), nil
	}

	return base.ResolveReference(target).String(), nil
}

func convertWebMToGIF(ffmpegPath, inputPath, outputPath string, fps, scaleWidth, maxColors int) error {
	filter := fmt.Sprintf(
		"fps=%d,scale=%d:-1:flags=lanczos,split[s0][s1];[s0]palettegen=max_colors=%d:reserve_transparent=0[p];[s1][p]paletteuse=dither=none:diff_mode=rectangle",
		fps,
		scaleWidth,
		maxColors,
	)
	cmd := exec.Command(
		ffmpegPath,
		"-y",
		"-loglevel", "error",
		"-i", inputPath,
		"-vf", filter,
		"-loop", "0",
		outputPath,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("convert webm to gif: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func optimizeGIFWithGifsicle(gifsiclePath, gifPath string, maxColors, lossy int) error {
	colors := strconv.Itoa(maxColors)
	lossyFlag := fmt.Sprintf("--lossy=%d", lossy)

	first := exec.Command(
		gifsiclePath,
		"-O3",
		"--colors", colors,
		lossyFlag,
		gifPath,
		"-o", gifPath,
	)
	if output, err := first.CombinedOutput(); err == nil {
		return nil
	} else if err := runGifsicleFallback(gifsiclePath, gifPath, colors); err != nil {
		return fmt.Errorf("optimize gif with gifsicle: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func runGifsicleFallback(gifsiclePath, gifPath, colors string) error {
	fallback := exec.Command(
		gifsiclePath,
		"-O3",
		"--colors", colors,
		gifPath,
		"-o", gifPath,
	)
	if output, err := fallback.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
