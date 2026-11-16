# Runbook: rotar secretos

Cómo cambiar las llaves de la API sin cerrar sesiones ni perder datos (T38). Rote una llave cuando se sospeche que se filtró, cuando salga del equipo alguien que la conocía, o como rutina (una vez al año).

Las llaves viven en Key Vault (T45) y la API las lee al arrancar. Generar una llave: `openssl rand -base64 48` para `JWT_SECRET` y `openssl rand -base64 32` para `INVOICING_TOKEN_KEY`. Nunca en el repo, en un chat ni en un log.

## `JWT_SECRET` (firma los access tokens)

Los access tokens duran 15 minutos.

1. Copie la llave actual a `JWT_SECRET_PREVIOUS` y ponga la nueva en `JWT_SECRET`.
2. Despliegue. Los tokens nuevos se firman con la nueva llave y los anteriores siguen sirviendo hasta que venzan.
3. Pasados 15 minutos (o en el siguiente despliegue), borre `JWT_SECRET_PREVIOUS`.

**Si la llave se filtró:** sáltese el paso de la llave anterior. Los usuarios reciben un 401 y la app renueva la sesión sola con su refresh token, que no depende de esta llave.

## `INVOICING_TOKEN_KEY` (cifra el token de cada empresa en el proveedor de facturación)

1. Copie la llave actual a `INVOICING_TOKEN_KEY_PREVIOUS` y ponga la nueva en `INVOICING_TOKEN_KEY`.
2. Despliegue. Cada token se abre con la llave anterior la próxima vez que se lee (al emitir un documento o al consultar la cuenta del proveedor) y se guarda cifrado con la nueva.
3. Antes de borrar `INVOICING_TOKEN_KEY_PREVIOUS`, verifique que no quedan tokens con la llave anterior: con la llave nueva sola, `GET /api/v1/invoicing/provider` de cada empresa con cuenta debe responder sin error. Un token que no abre aparece en los logs como `stored token does not open with this key`.
4. Borre `INVOICING_TOKEN_KEY_PREVIOUS` y despliegue.

**Si la llave se filtró:** además de rotarla, pida al proveedor tecnológico que rote los tokens de las empresas afectadas y regístrelos de nuevo.

## Contraseña de la base de datos (`APP_DB_PASSWORD`)

La fija `cmd/migrate` en cada despliegue. Cámbiela en Key Vault y despliegue: las migraciones actualizan la contraseña de `app_user` y la API se reconecta con la nueva.
