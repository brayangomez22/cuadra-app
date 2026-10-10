package domain

import (
	"time"

	"github.com/google/uuid"
)

const maxCategoryNameLength = 100

// Category groups products in a simple tree: a category has an optional
// parent. The domain only rejects a category being its own parent; deeper
// cycles (A → B → A) need the stored tree, so the use case that moves a
// category checks them.
type Category struct {
	id        uuid.UUID
	tenantID  uuid.UUID
	name      string
	parentID  *uuid.UUID
	createdAt time.Time
}

// NewCategory creates a category; parentID nil makes it a root category.
func NewCategory(tenantID uuid.UUID, name string, parentID *uuid.UUID, now time.Time) (*Category, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	return newCategory(CategorySnapshot{ID: id, TenantID: tenantID, Name: name, ParentID: parentID, CreatedAt: now})
}

// newCategory checks the invariants shared by NewCategory and RehydrateCategory.
func newCategory(s CategorySnapshot) (*Category, error) {
	if s.ID == uuid.Nil {
		return nil, ErrInvalidCategoryID
	}
	if s.TenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	name, err := normalizeText(s.Name, maxCategoryNameLength, ErrInvalidCategoryName)
	if err != nil {
		return nil, err
	}
	c := &Category{id: s.ID, tenantID: s.TenantID, name: name, createdAt: s.CreatedAt.UTC()}
	if err := c.MoveTo(s.ParentID); err != nil {
		return nil, err
	}
	return c, nil
}

// ID returns the category ID (UUID v7).
func (c *Category) ID() uuid.UUID { return c.id }

// TenantID returns the tenant the category belongs to.
func (c *Category) TenantID() uuid.UUID { return c.tenantID }

// Name returns the category name.
func (c *Category) Name() string { return c.name }

// ParentID returns a copy of the parent's ID, or nil for a root category.
func (c *Category) ParentID() *uuid.UUID { return copyID(c.parentID) }

// CreatedAt returns the creation time in UTC.
func (c *Category) CreatedAt() time.Time { return c.createdAt }

// Rename changes the name; on error the name is unchanged.
func (c *Category) Rename(name string) error {
	name, err := normalizeText(name, maxCategoryNameLength, ErrInvalidCategoryName)
	if err != nil {
		return err
	}
	c.name = name
	return nil
}

// MoveTo changes the parent; nil makes the category a root. It rejects a nil
// UUID and the category itself as parent, leaving the parent unchanged.
func (c *Category) MoveTo(parentID *uuid.UUID) error {
	if parentID != nil && (*parentID == uuid.Nil || *parentID == c.id) {
		return ErrInvalidCategoryParent
	}
	c.parentID = copyID(parentID)
	return nil
}

// CategorySnapshot holds a stored category's fields, for RehydrateCategory.
type CategorySnapshot struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	Name      string
	ParentID  *uuid.UUID
	CreatedAt time.Time
}

// RehydrateCategory rebuilds a stored category, checking the same invariants
// as NewCategory so corrupt data surfaces as an error instead of an invalid entity.
func RehydrateCategory(s CategorySnapshot) (*Category, error) { return newCategory(s) }
