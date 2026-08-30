package handler

import (
	_ "embed"
	"net/http"
)

// Embed de las paginas legales al binario. Sirve para tener URLs publicas
// requeridas por Play Store y para que la app pueda linkearlas.
//
//go:embed static/privacy.html
var privacyHTML []byte

//go:embed static/terms.html
var termsHTML []byte

//go:embed static/delete-account.html
var deleteAccountHTML []byte

// PrivacyPolicyHandler sirve la politica de privacidad en /privacy.
func PrivacyPolicyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(privacyHTML)
}

// TermsHandler sirve los terminos y condiciones en /terms.
func TermsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(termsHTML)
}

// DeleteAccountPageHandler sirve la pagina publica de instrucciones para
// eliminar la cuenta. Requerida por Play Store desde 2023.
func DeleteAccountPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(deleteAccountHTML)
}
