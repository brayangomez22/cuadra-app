package domain

// The rules of managing users. Who may manage users at all is the
// users:manage permission, checked before the use case runs; these rules
// protect the owners: only an owner manages owners, and a tenant always
// keeps an active one.

// CheckAssignable reports whether a user with role actor may give role r:
// ErrOwnerOnly if r is owner and actor is not.
func CheckAssignable(actor, r Role) error {
	if !r.valid() {
		return ErrInvalidRole
	}
	if r == RoleOwner && actor != RoleOwner {
		return ErrOwnerOnly
	}
	return nil
}

// CheckManages reports whether actor may change target (its role, its
// state or its password): ErrOwnerOnly if target is an owner and actor is
// not.
func CheckManages(actor, target *User) error {
	if target.role == RoleOwner && actor.role != RoleOwner {
		return ErrOwnerOnly
	}
	return nil
}

// ChangeRoleBy gives the user role r on behalf of actor. activeOwners is
// how many active owners the tenant has: the last one cannot stop being
// an owner (ErrLastOwner). On error the role is unchanged.
func (u *User) ChangeRoleBy(actor *User, r Role, activeOwners int) error {
	if err := CheckAssignable(actor.role, r); err != nil {
		return err
	}
	if err := CheckManages(actor, u); err != nil {
		return err
	}
	if u.isLastOwner(activeOwners) && r != RoleOwner {
		return ErrLastOwner
	}
	u.role = r
	return nil
}

// DeactivateBy blocks the user on behalf of actor. Nobody deactivates
// themselves (ErrSelfDeactivation), and the last active owner stays
// (ErrLastOwner).
func (u *User) DeactivateBy(actor *User, activeOwners int) error {
	if u.id == actor.id {
		return ErrSelfDeactivation
	}
	if err := CheckManages(actor, u); err != nil {
		return err
	}
	if u.isLastOwner(activeOwners) {
		return ErrLastOwner
	}
	u.active = false
	return nil
}

// ActivateBy reactivates the user on behalf of actor.
func (u *User) ActivateBy(actor *User) error {
	if err := CheckManages(actor, u); err != nil {
		return err
	}
	u.active = true
	return nil
}

// SetPasswordHash replaces the password; hash must come from a
// PasswordHasher, after ValidatePassword.
func (u *User) SetPasswordHash(hash string) error {
	if hash == "" {
		return ErrEmptyPasswordHash
	}
	u.passwordHash = hash
	return nil
}

func (u *User) isLastOwner(activeOwners int) bool {
	return u.role == RoleOwner && u.active && activeOwners <= 1
}
