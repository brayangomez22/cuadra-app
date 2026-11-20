package domain

import "errors"

// Errors of the getting-started checklist.
var (
	// ErrNoSteps: a template declares no step.
	ErrNoSteps = errors.New("onboarding: a template must declare its getting-started steps")
	// ErrUnknownStep: a template names a step outside the catalog.
	ErrUnknownStep = errors.New("onboarding: unknown getting-started step")
	// ErrRepeatedStep: a template names a step twice.
	ErrRepeatedStep = errors.New("onboarding: repeated getting-started step")
	// ErrNothingToDismiss: the user has no step to hide.
	ErrNothingToDismiss = errors.New("onboarding: the checklist has no step to hide")
)
