package templates

import "fmt"

// Checks are the validations of a template that need what lives outside
// this package: the module registry and the getting-started steps of the
// onboarding module. internal/server gives them.
type Checks struct {
	// Modules validates the template's modules (the registry's CheckSet).
	Modules func(ids []string) error
	// Steps validates the template's getting-started steps.
	Steps func(ids []string) error
}

// check runs c on t.
func (c Checks) check(t Template) error {
	if err := c.Modules(t.Modules); err != nil {
		return fmt.Errorf("templates: %s: %w", t.ID, err)
	}
	if err := c.Steps(t.GettingStarted); err != nil {
		return fmt.Errorf("templates: %s: getting_started: %w", t.ID, err)
	}
	return nil
}
