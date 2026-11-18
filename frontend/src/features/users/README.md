# users

La gestión de usuarios de la empresa (T41), solo para dueño y administrador (`users:manage`).

- **Pantalla:** `/usuarios` (`UsersPage`): los usuarios con su rol y estado, y las invitaciones sin aceptar.
- **Invitar:** crea una invitación y muestra su enlace (`/invitacion#<token>`) para compartirlo; Cuadra todavía no lo envía por correo. Invitar otra vez el mismo correo anula el enlace anterior.
- **Por usuario:** cambiar el rol, generar el enlace para restablecer la contraseña (`/restablecer#<token>`), desactivar (pierde el acceso en su siguiente petición) y reactivar.
- **Reglas** (`model/userActions.ts`, las mismas de la API): solo un dueño gestiona a otro dueño o da ese rol; nadie se desactiva a sí mismo. Que la empresa conserve un dueño activo lo responde la API (`last_owner`).
- **API pública** (`index.ts`): `useUsers` y el tipo `User`, para las pantallas que nombran usuarios (la auditoría).

Las páginas para aceptar la invitación y restablecer la contraseña son de `auth`, porque las abre alguien sin sesión.
