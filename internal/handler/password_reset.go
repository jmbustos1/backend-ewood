package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"ewood-backend/internal/client/postgres"
	"ewood-backend/internal/client/ses"
	"ewood-backend/internal/models"
)

const (
	tokenBytes         = 32               // 32 bytes -> 64 chars hex
	resetTokenValidity = 30 * time.Minute
)

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// ForgotPasswordHandler: recibe un email, si existe genera un token unico,
// lo persiste en password_resets, y manda un email con el link de reset.
//
// Siempre responde 200 con el mismo mensaje generico (no revela si el email
// existe o no — evita email enumeration attack).
func ForgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email required")
		return
	}

	// Respuesta generica se envia al final. Toda la logica va async por si el
	// email tarda — pero acá lo hacemos sync para poder loggear errores en
	// desarrollo. En prod se puede pasar a goroutine.
	genericResponse := map[string]string{
		"message": "si el email existe en nuestra base, enviaremos las instrucciones para recuperar la contrasena",
	}

	var user models.User
	err := postgres.GDB.Where("email = ? AND deleted_at IS NULL", req.Email).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Email no existe. Respondemos igual para no revelar la ausencia.
			writeJSON(w, http.StatusOK, genericResponse)
			return
		}
		log.Printf("forgot: db lookup failed: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Generar token criptograficamente seguro
	tokenBytes := make([]byte, tokenBytes)
	if _, err := rand.Read(tokenBytes); err != nil {
		log.Printf("forgot: token gen failed: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	token := hex.EncodeToString(tokenBytes)

	// Persistir en DB. Los tokens viejos del mismo user quedan (queryeamos por
	// token, no por user_id), pero si querés puedes invalidarlos aca.
	reset := models.PasswordReset{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(resetTokenValidity),
	}
	if err := postgres.GDB.Create(&reset).Error; err != nil {
		log.Printf("forgot: db insert failed: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Construir la URL del link. El backend expone /reset-password?token=XXX
	// como fallback HTML, y la app captura el deep link ewood://reset-password?token=XXX.
	// El email lleva el HTTPS: si tienen la app instalada, Android/iOS abren
	// la app automaticamente por associated domain (a configurar). Si no,
	// se abre el fallback web.
	backendURL := os.Getenv("MP_BACKEND_BASE_URL")
	if backendURL == "" {
		backendURL = "https://api.exwood.dev"
	}
	resetURL := backendURL + "/reset-password?token=" + token

	sesClient, err := ses.New(context.Background())
	if err != nil {
		log.Printf("forgot: ses init failed: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	messageID, err := sesClient.SendEmail(
		context.Background(),
		user.Email,
		"Recupera tu contrasena — Exwood",
		passwordResetEmailHTML(user.Name, resetURL),
		passwordResetEmailText(user.Name, resetURL),
	)
	if err != nil {
		log.Printf("forgot: ses send failed for %s: %v", user.Email, err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	log.Printf("forgot: sent reset email to %s (message_id=%s)", user.Email, messageID)

	writeJSON(w, http.StatusOK, genericResponse)
}

// ResetPasswordHandler: valida el token, actualiza el password_hash del user,
// y marca el token como usado. Los tokens son one-shot.
func ResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Token == "" || len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "token and new_password (min 8 chars) are required")
		return
	}

	var reset models.PasswordReset
	err := postgres.GDB.
		Where("token = ? AND used_at IS NULL AND expires_at > ?", req.Token, time.Now()).
		First(&reset).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			writeError(w, http.StatusBadRequest, "invalid or expired token")
			return
		}
		log.Printf("reset: db lookup failed: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	newHashStr := string(newHash)

	// Actualizar el password + marcar el token como usado en una transaccion.
	tx := postgres.GDB.Begin()
	if err := tx.Model(&models.User{}).Where("id = ?", reset.UserID).
		Update("password_hash", &newHashStr).Error; err != nil {
		tx.Rollback()
		log.Printf("reset: update user failed: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	now := time.Now()
	if err := tx.Model(&reset).Update("used_at", &now).Error; err != nil {
		tx.Rollback()
		log.Printf("reset: mark used failed: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := tx.Commit().Error; err != nil {
		log.Printf("reset: commit failed: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	log.Printf("reset: password updated for user_id=%d", reset.UserID)
	writeJSON(w, http.StatusOK, map[string]string{"message": "password actualizada"})
}
