// Package web holds the page that builds a card's URL, served at /, and the
// API docs page, served at /docs.
package web

import "embed"

// FS holds the pages, their scripts and styles, and the favicon.
//
//go:embed index.html app.js style.css docs.html docs.js docs.css favicon.svg
var FS embed.FS
