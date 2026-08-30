package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"ewood-backend/internal/client/postgres"
	"ewood-backend/internal/middleware"
	"ewood-backend/internal/models"
)

// ─── Request / Response types ───────────────────────────────────────────────

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`

	// Consentimiento a documentos legales. Opcionales para no romper apps
	// viejas — cuando la app envie estos campos, quedan persistidos en la
	// tabla user_consents. Si no vienen, se asume aceptacion implicita
	// (checkbox aceptado en la app) sin registro versionado.
	// Ver ESTADO-ACTUAL.md P1 #9.
	AcceptedTermsVersion   string `json:"accepted_terms_version,omitempty"`
	AcceptedPrivacyVersion string `json:"accepted_privacy_version,omitempty"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type authResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	User         userPayload `json:"user"`
}

type userPayload struct {
	ID              uint   `json:"id"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	PreferredCityID *uint  `json:"preferred_city_id,omitempty"`
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func jwtSecret() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		s = "ewood-secret-change-in-production"
	}
	return []byte(s)
}

func generateTokens(userID uint) (access, refresh string, err error) {
	now := time.Now()

	accessClaims := jwt.MapClaims{
		"user_id": userID,
		"exp":     now.Add(24 * time.Hour).Unix(),
		"iat":     now.Unix(),
		"type":    "access",
	}
	access, err = jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(jwtSecret())
	if err != nil {
		return
	}

	refreshClaims := jwt.MapClaims{
		"user_id": userID,
		"exp":     now.Add(30 * 24 * time.Hour).Unix(),
		"iat":     now.Unix(),
		"type":    "refresh",
	}
	refresh, err = jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(jwtSecret())
	return
}

func toUserPayload(u models.User) userPayload {
	return userPayload{ID: u.ID, Name: u.Name, Email: u.Email, PreferredCityID: u.PreferredCityID}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// ─── Handlers ─────────────────────────────────────────────────────────────────

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.Email == "" || len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "name, email and password (min 8 chars) are required")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	hashStr := string(hash)
	user := models.User{Name: req.Name, Email: req.Email, PasswordHash: &hashStr}
	if err := postgres.GDB.Create(&user).Error; err != nil {
		writeError(w, http.StatusConflict, "email already in use")
		return
	}

	// Persistir consentimiento a documentos legales si vinieron en el request.
	// Best-effort: si falla, no rollbackeamos el user creado (el consent es
	// una mejora de trazabilidad, no un requisito duro para el registro).
	persistUserConsent(user.ID, ConsentTermsDoc, req.AcceptedTermsVersion, r)
	persistUserConsent(user.ID, ConsentPrivacyDoc, req.AcceptedPrivacyVersion, r)

	access, refresh, err := generateTokens(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate tokens")
		return
	}

	writeJSON(w, http.StatusCreated, authResponse{AccessToken: access, RefreshToken: refresh, User: toUserPayload(user)})
}

// Constantes de tipo documento — se copian aca para que el handler no dependa
// del paquete models (ya usa models.User pero por consistencia).
const (
	ConsentTermsDoc   = "terms"
	ConsentPrivacyDoc = "privacy"
)

// persistUserConsent crea una fila en user_consents si version != "". Silencioso
// ante fallos (best-effort).
func persistUserConsent(userID uint, document, version string, r *http.Request) {
	if version == "" {
		return
	}
	// IP: prioriza X-Forwarded-For (Caddy) o cae a RemoteAddr.
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.RemoteAddr
	}
	if len(ip) > 64 {
		ip = ip[:64]
	}
	ua := r.UserAgent()
	if len(ua) > 255 {
		ua = ua[:255]
	}

	consent := models.UserConsent{
		UserID:     userID,
		Document:   document,
		Version:    version,
		AcceptedAt: time.Now(),
		IPAddress:  ip,
		UserAgent:  ua,
	}
	// Ignorar error: best-effort.
	_ = postgres.GDB.Create(&consent).Error
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var user models.User
	if err := postgres.GDB.Where("email = ? AND deleted_at IS NULL", req.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if user.PasswordHash == nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	access, refresh, err := generateTokens(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate tokens")
		return
	}

	writeJSON(w, http.StatusOK, authResponse{AccessToken: access, RefreshToken: refresh, User: toUserPayload(user)})
}

func RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	token, err := jwt.Parse(req.RefreshToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return jwtSecret(), nil
	})
	if err != nil || !token.Valid {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["type"] != "refresh" {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	userID := uint(claims["user_id"].(float64))
	access, refresh, err := generateTokens(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate tokens")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"access_token": access, "refresh_token": refresh})
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Stateless JWT: el cliente elimina sus tokens. Aquí podemos blacklistear en futuro.
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func MeHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var user models.User
	if err := postgres.GDB.First(&user, userID).Error; err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, toUserPayload(user))
}

// ForgotPasswordHandler y ResetPasswordHandler se implementan en password_reset.go
// (SES + tokens en tabla password_resets).
