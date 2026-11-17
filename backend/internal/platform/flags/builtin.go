package flags

// Builtin returns the flags the product declares. A new flag is added here
// (and removed once its feature is released to everyone).
func Builtin() []Definition {
	return []Definition{
		{
			Key:         "analytics",
			Description: "Envía la analítica de uso de la app (solo si la empresa la aceptó).",
		},
		{
			Key:         "feedback",
			Description: "Muestra el botón «Danos tu opinión» en la app.",
		},
	}
}
