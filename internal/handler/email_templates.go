package handler

import "fmt"

// passwordResetEmailHTML retorna el body HTML del email de reset password.
// El link va tanto al deep-link app (ewood://reset-password?token=...) como
// al fallback web (https://api.exwood.dev/reset-password?token=...).
func passwordResetEmailHTML(userName, resetURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>Recupera tu contrasena — Exwood</title>
</head>
<body style="font-family:-apple-system,BlinkMacSystemFont,sans-serif;background:#f5f5f5;margin:0;padding:20px;">
  <div style="max-width:520px;margin:0 auto;background:#fff;border-radius:16px;padding:32px;">
    <h1 style="color:#111;font-size:24px;margin:0 0 16px;">Recupera tu contrasena</h1>
    <p style="color:#555;line-height:1.6;">Hola %s,</p>
    <p style="color:#555;line-height:1.6;">
      Recibimos una solicitud para restablecer la contrasena de tu cuenta en Exwood.
      Toca el boton para elegir una nueva contrasena.
    </p>
    <p style="text-align:center;margin:32px 0;">
      <a href="%s"
         style="background:#4DB300;color:#fff;padding:14px 32px;border-radius:24px;
                text-decoration:none;font-weight:bold;display:inline-block;">
        Cambiar contrasena
      </a>
    </p>
    <p style="color:#888;font-size:13px;line-height:1.5;">
      Este link expira en <strong>30 minutos</strong>. Si no fuiste tu quien solicito
      el cambio, ignora este email — tu contrasena no cambiara.
    </p>
    <hr style="border:none;border-top:1px solid #eee;margin:24px 0;">
    <p style="color:#aaa;font-size:12px;margin:0;">
      Si el boton no funciona, copia y pega esta URL en tu navegador:<br>
      <span style="word-break:break-all;">%s</span>
    </p>
  </div>
  <p style="text-align:center;color:#aaa;font-size:12px;margin-top:24px;">
    Exwood · Movilidad electrica urbana · contacto@exwood.cl
  </p>
</body>
</html>`, userName, resetURL, resetURL)
}

// passwordResetEmailText retorna la version plain text (fallback si el cliente
// no renderiza HTML).
func passwordResetEmailText(userName, resetURL string) string {
	return fmt.Sprintf(`Hola %s,

Recibimos una solicitud para restablecer la contrasena de tu cuenta en Exwood.

Usa este link para elegir una nueva contrasena:
%s

El link expira en 30 minutos. Si no solicitaste el cambio, ignora este email.

—
Exwood
contacto@exwood.cl
`, userName, resetURL)
}
