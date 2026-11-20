package domain

import "slices"

// StepState is a step of a user's checklist.
type StepState struct {
	ID       StepID
	Done     bool
	Critical bool
}

// Checklist is the getting-started checklist of a user: the steps of the
// tenant's template that concern the tenant and the user can resolve.
type Checklist struct {
	Steps []StepState
	// Dismissed is whether the user hid it.
	Dismissed bool
	// back is whether a critical step went missing after it was hidden.
	back bool
}

// ChecklistInput is what a checklist is built from.
type ChecklistInput struct {
	// Steps are the template's, in its order (already validated).
	Steps []string
	Facts Facts
	// Can is whether the user's role has a permission.
	Can func(permission string) bool
	// Modules are the tenant's active modules.
	Modules []string
	// Dismissal is the user's, if they hid the checklist.
	Dismissal *Dismissal
}

// BuildChecklist builds a user's checklist. A step is left out when it
// does not apply to the tenant, its module is off or the user's role cannot
// resolve it.
func BuildChecklist(in ChecklistInput) Checklist {
	c := Checklist{Dismissed: in.Dismissal != nil}
	for _, id := range in.Steps {
		s, ok := stepByID(StepID(id))
		if !ok || !s.concerns(in) {
			continue
		}
		state := StepState{ID: s.ID, Done: s.done(in.Facts), Critical: s.Critical}
		c.Steps = append(c.Steps, state)
		if in.Dismissal != nil && state.Critical && !state.Done && !slices.Contains(in.Dismissal.pending, s.ID) {
			c.back = true
		}
	}
	return c
}

// concerns is whether s belongs in the checklist of in.
func (s Step) concerns(in ChecklistInput) bool {
	if s.applies != nil && !s.applies(in.Facts) {
		return false
	}
	if s.Module != "" && !slices.Contains(in.Modules, s.Module) {
		return false
	}
	return in.Can(s.Permission)
}

// Total is how many steps the checklist has.
func (c Checklist) Total() int { return len(c.Steps) }

// Completed is how many of them are done.
func (c Checklist) Completed() int {
	n := 0
	for _, s := range c.Steps {
		if s.Done {
			n++
		}
	}
	return n
}

// Visible is whether the app shows the checklist: something is left to do
// and the user has not hidden it, or a critical step went missing after
// they did (a resolution that expired, invoicing turned on without one).
func (c Checklist) Visible() bool {
	if c.Completed() == c.Total() {
		return false
	}
	return !c.Dismissed || c.back
}

// pending returns the steps not done yet.
func (c Checklist) pending() []StepID {
	var ids []StepID
	for _, s := range c.Steps {
		if !s.Done {
			ids = append(ids, s.ID)
		}
	}
	return ids
}
