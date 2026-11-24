// Package web holds the page that builds a card's URL. It is served at /.
package web

import "embed"

// FS holds index.html, app.js and style.css.
//
//go:embed index.html app.js style.css
var FS embed.FS
