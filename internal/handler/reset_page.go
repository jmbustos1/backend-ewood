package handler

import (
	"net/http"
)

// ResetPasswordPageHandler devuelve un formulario HTML simple para que el
// usuario elija nueva contrasena. Es fallback web para cuando el link del
// email no engancha con el deep link de la app (ej. lo abre desde desktop).
//
// El form hace POST a /auth/reset-password con { token, new_password }.
func ResetPasswordPageHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "token missing", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="es">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>Recupera tu contrasena — Ewood</title>
  <style>
    :root { color-scheme: dark; }
    body { font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;
           background:#0A0A0A;color:#fff;margin:0;padding:24px;
           display:flex;min-height:100vh;align-items:center;justify-content:center; }
    .card { background:#1A1A1A;border-radius:16px;padding:32px;max-width:420px;width:100%; }
    h1 { margin:0 0 8px;font-size:26px;font-weight:700; }
    p.sub { color:rgba(255,255,255,0.6);margin:0 0 24px;font-size:14px; }
    label { display:block;font-size:12px;text-transform:uppercase;
            letter-spacing:1px;color:rgba(255,255,255,0.6);margin-bottom:6px; }
    input[type=password] { width:100%;box-sizing:border-box;
                            background:#262626;border:none;border-radius:8px;
                            padding:14px;color:#fff;font-size:16px;
                            margin-bottom:16px; }
    button { width:100%;background:#B4F33C;color:#0A0A0A;border:none;
             border-radius:24px;padding:16px;font-size:15px;font-weight:700;
             cursor:pointer; }
    button:disabled { background:#262626;color:rgba(255,255,255,0.3);cursor:not-allowed; }
    .err { color:#EF4444;font-size:13px;margin-top:8px; }
    .ok { color:#4DB300;font-size:14px;margin-top:16px;text-align:center; }
  </style>
</head>
<body>
  <div class="card">
    <h1>Nueva contrasena</h1>
    <p class="sub">Elige una contrasena de al menos 8 caracteres.</p>
    <form id="f">
      <label>Nueva contrasena</label>
      <input type="password" id="p1" required minlength="8" autocomplete="new-password">
      <label>Repite la contrasena</label>
      <input type="password" id="p2" required minlength="8" autocomplete="new-password">
      <button type="submit" id="submit">Guardar contrasena</button>
      <p id="msg" class="err" style="display:none;"></p>
      <p id="okmsg" class="ok" style="display:none;"></p>
    </form>
  </div>
<script>
const token = new URLSearchParams(location.search).get('token');
const f = document.getElementById('f');
const msg = document.getElementById('msg');
const okmsg = document.getElementById('okmsg');
const btn = document.getElementById('submit');
f.addEventListener('submit', async e => {
  e.preventDefault();
  msg.style.display = 'none';
  const p1 = document.getElementById('p1').value;
  const p2 = document.getElementById('p2').value;
  if (p1 !== p2) {
    msg.textContent = 'Las contrasenas no coinciden';
    msg.style.display = 'block';
    return;
  }
  btn.disabled = true;
  btn.textContent = 'Guardando...';
  try {
    const r = await fetch('/auth/reset-password', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({ token, new_password: p1 }),
    });
    const data = await r.json();
    if (!r.ok) throw new Error(data.error || 'Error');
    okmsg.textContent = 'Listo. Ya puedes iniciar sesion desde la app.';
    okmsg.style.display = 'block';
    f.querySelectorAll('input').forEach(i => i.disabled = true);
    btn.style.display = 'none';
  } catch (err) {
    msg.textContent = err.message;
    msg.style.display = 'block';
    btn.disabled = false;
    btn.textContent = 'Guardar contrasena';
  }
});
</script>
</body>
</html>`))
}
