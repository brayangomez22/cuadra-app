// Package business embeds the business templates (*.yaml): the data a new
// tenant starts with, by kind of business. internal/platform/templates
// validates and reads them; a template is added by adding a file here.
package business

import "embed"

// FS holds the templates, one per file; the file name is the template's ID.
//
//go:embed *.yaml
var FS embed.FS
