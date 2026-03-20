package graphiti

import "embed"

//go:embed web/static/*
var staticFS embed.FS

//go:embed config/nodes/*
var defaultNodeDefs embed.FS
