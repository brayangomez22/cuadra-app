package domain

import "strings"

// commonPasswords are passwords of at least MinPasswordLength characters
// from the lists of the most used ones (and of the leaked ones in
// Colombia), lowercase: the first an attacker tries. The whole password
// must match; a phrase that contains one is fine.
var commonPasswords = map[string]bool{
	"12345678": true, "123456789": true, "1234567890": true, "12345678910": true, "87654321": true,
	"11111111": true, "00000000": true, "12341234": true, "11223344": true, "123123123": true,
	"password": true, "password1": true, "password123": true, "passw0rd": true, "p@ssw0rd": true,
	"qwerty123": true, "qwertyuiop": true, "asdfghjkl": true, "1q2w3e4r": true, "1qaz2wsx": true,
	"abcd1234": true, "abc12345": true, "aa123456": true, "iloveyou": true, "sunshine": true,
	"princess": true, "football": true, "baseball": true, "superman": true, "starwars": true,
	"welcome1": true, "trustno1": true, "whatever": true, "computer": true, "michelle": true,
	"contraseña": true, "contraseña1": true, "contraseña123": true, "contrasena": true, "contrasena123": true,
	"clave123": true, "clave1234": true, "miclave123": true, "123456789a": true, "colombia": true,
	"colombia1": true, "colombia123": true, "bogota123": true, "medellin": true, "medellin123": true,
	"teamo123": true, "tequiero": true, "mariposa": true, "princesa": true, "estrella": true,
	"america1": true, "nacional": true, "millonarios": true, "barcelona": true, "realmadrid": true,
	"admin123": true, "administrador": true, "usuario123": true, "cuadra123": true, "cuadra2026": true,
	"inventario": true, "tienda123": true, "negocio123": true, "empresa123": true, "ventas123": true,
}

// isCommonPassword tells whether plain, ignoring case, is in the list.
func isCommonPassword(plain string) bool { return commonPasswords[strings.ToLower(plain)] }
