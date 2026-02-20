package routes

//go:generate go run ../../tools/routing/cmd/route-codegen/main.go -config ../../routes.yaml -output routes_gen.go -package routes -paths-output paths/paths_gen.go -paths-package paths -static-dir static -locale-package github.com/Y4shin/conference-tool/internal/locale
