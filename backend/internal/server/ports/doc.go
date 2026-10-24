// Package ports joins the modules: each adapter satisfies the port one
// module declares in its app/ package with another module, and translates
// its errors, so no module imports another. Only internal/server uses it.
package ports
