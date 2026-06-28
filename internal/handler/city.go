package handler

import (
	"net/http"

	"ewood-backend/internal/client/postgres"
	"ewood-backend/internal/models"
)

func ListCitiesHandler(w http.ResponseWriter, r *http.Request) {
	var cities []models.City
	if err := postgres.GDB.Where("active = true AND deleted_at IS NULL").Find(&cities).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not fetch cities")
		return
	}
	writeJSON(w, http.StatusOK, cities)
}
