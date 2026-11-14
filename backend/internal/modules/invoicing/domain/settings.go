package domain

// Settings are a tenant's choices about electronic documents. Not every
// business must issue them, so emission starts off.
type Settings struct {
	// Enabled is whether confirmed sales get an electronic document.
	Enabled bool
	// DefaultKind is the kind of the sales that ask for none.
	DefaultKind Kind
}

// DefaultSettings are the settings of a tenant that never changed them:
// emission off, point of sale documents.
func DefaultSettings() Settings { return Settings{DefaultKind: KindPOS} }

// NewSettings validates the settings: ErrInvalidKind for an unknown kind.
func NewSettings(enabled bool, kind Kind) (Settings, error) {
	k, err := ParseKind(string(kind))
	if err != nil {
		return Settings{}, err
	}
	return Settings{Enabled: enabled, DefaultKind: k}, nil
}
