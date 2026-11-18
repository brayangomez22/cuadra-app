package httpadapter

import (
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// User-facing messages written by the handlers themselves.
const (
	invalidSessionMessage  = "Su sesión expiró o no es válida. Inicie sesión de nuevo."
	unauthorizedMessage    = "Su sesión no es válida o expiró. Inicie sesión de nuevo."
	tooManyRequestsMessage = "Hizo demasiados intentos. Espere un momento antes de volver a intentar."
	tenantSelectionMessage = "Su cuenta existe en varias empresas. Elija a cuál quiere entrar."
	wrongLoginMessage      = "El correo o la contraseña no son correctos."
)

// errorRules maps the identity domain errors to HTTP. Session errors
// (refresh) and rate limits are answered by the handlers, which also set
// headers.
var errorRules = []httpx.ErrorRule{
	{Err: domain.ErrInvalidCredentials, Status: http.StatusUnauthorized, Code: "invalid_credentials", Message: wrongLoginMessage},
	{Err: domain.ErrTenantSuspended, Status: http.StatusForbidden, Code: "tenant_suspended", Message: "Esta empresa está suspendida. Comuníquese con soporte."},
	{Err: domain.ErrUserNotFound, Status: http.StatusUnauthorized, Code: "unauthorized", Message: unauthorizedMessage},
	{Err: domain.ErrInvalidEmail, Status: http.StatusUnprocessableEntity, Code: "invalid_email", Message: "El correo no es válido."},
	{Err: domain.ErrInvalidNITCheckDigit, Status: http.StatusUnprocessableEntity, Code: "invalid_nit_check_digit", Message: "El dígito de verificación del NIT no coincide."},
	{Err: domain.ErrInvalidNIT, Status: http.StatusUnprocessableEntity, Code: "invalid_nit", Message: "El NIT no es válido. Escríbalo con su dígito de verificación, por ejemplo 900123456-8."},
	{Err: domain.ErrPasswordTooShort, Status: http.StatusUnprocessableEntity, Code: "password_too_short", Message: "La contraseña debe tener al menos 8 caracteres."},
	{Err: domain.ErrPasswordTooLong, Status: http.StatusUnprocessableEntity, Code: "password_too_long", Message: "La contraseña puede tener máximo 72 caracteres."},
	{Err: domain.ErrPasswordTooCommon, Status: http.StatusUnprocessableEntity, Code: "password_too_common", Message: "Esa contraseña es de las más usadas y fácil de adivinar. Elija otra, por ejemplo una frase."},
	{Err: domain.ErrInvalidTenantName, Status: http.StatusUnprocessableEntity, Code: "invalid_tenant_name", Message: "Escriba el nombre de la empresa (máximo 200 caracteres)."},
	{Err: domain.ErrUnknownBusinessTemplate, Status: http.StatusUnprocessableEntity, Code: "unknown_business_template", Message: "Elija uno de los tipos de negocio disponibles."},
	{Err: domain.ErrInvalidDefaultTaxRate, Status: http.StatusUnprocessableEntity, Code: "invalid_default_tax_rate", Message: "El IVA por defecto debe ser un porcentaje de 0 a menos de 100, con hasta 2 decimales."},
	{Err: domain.ErrInvalidTenantAddress, Status: http.StatusUnprocessableEntity, Code: "invalid_tenant_address", Message: "La dirección y la ciudad deben ir en una sola línea (máximo 200 y 100 caracteres)."},
	{Err: domain.ErrInvalidMunicipality, Status: http.StatusUnprocessableEntity, Code: "invalid_municipality", Message: "Elija un municipio de la lista del DANE."},
	{Err: domain.ErrInvalidTenantPhone, Status: http.StatusUnprocessableEntity, Code: "invalid_tenant_phone", Message: "Escriba el teléfono solo con números, espacios, +, - o paréntesis (máximo 30 caracteres)."},
	// A token whose tenant no longer exists is an invalid session, as for its user.
	{Err: domain.ErrTenantNotFound, Status: http.StatusUnauthorized, Code: "unauthorized", Message: unauthorizedMessage},
	{Err: domain.ErrDemoUnavailable, Status: http.StatusNotFound, Code: "demo_unavailable", Message: "La demostración no está disponible en este momento. Intente más tarde."},
	{Err: domain.ErrOwnerOnly, Status: http.StatusForbidden, Code: "owner_only", Message: "Solo un dueño puede dar el rol de dueño o cambiar a otro dueño."},
	{Err: domain.ErrLastOwner, Status: http.StatusConflict, Code: "last_owner", Message: "La empresa necesita al menos un dueño activo. Nombre otro dueño antes de hacer este cambio."},
	{Err: domain.ErrSelfDeactivation, Status: http.StatusConflict, Code: "self_deactivation", Message: "No puede desactivar su propio usuario."},
	{Err: domain.ErrEmailTaken, Status: http.StatusConflict, Code: "email_taken", Message: "Ese correo ya es de un usuario de la empresa."},
	{Err: domain.ErrInvalidRole, Status: http.StatusUnprocessableEntity, Code: "invalid_role", Message: "Elija uno de los roles disponibles."},
	{Err: domain.ErrInvitationNotFound, Status: http.StatusNotFound, Code: "not_found", Message: "La invitación no existe."},
	{Err: domain.ErrLinkNotFound, Status: http.StatusNotFound, Code: "link_not_found", Message: "El enlace no existe o está incompleto. Pida uno nuevo a quien se lo envió."},
	{Err: domain.ErrLinkExpired, Status: http.StatusGone, Code: "link_expired", Message: "El enlace venció. Pida uno nuevo a quien se lo envió."},
	{Err: domain.ErrLinkUsed, Status: http.StatusGone, Code: "link_used", Message: "El enlace ya se usó o fue anulado. Pida uno nuevo a quien se lo envió."},
	{Err: domain.ErrWrongPassword, Status: http.StatusUnprocessableEntity, Code: "wrong_password", Message: "La contraseña actual no es correcta."},
	{Err: domain.ErrInvalidUserName, Status: http.StatusUnprocessableEntity, Code: "invalid_user_name", Message: "Escriba su nombre (máximo 200 caracteres)."},
}
