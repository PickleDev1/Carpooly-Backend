package handlers

import (
	"car-backend/pkg/models"
	"car-backend/pkg/repository"
	"context"
	"encoding/json"
	"log"
	"fmt"
	"database/sql"
	"net/http"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type CarPoolRideHandler struct {
	carpoolRideRepo *repository.CarPoolRideRepository
}


func NewCarPoolRideHandler(repo *repository.CarPoolRideRepository) *CarPoolRideHandler {
	return &CarPoolRideHandler{
		carpoolRideRepo: repo,
	}
}

func (h *CarPoolRideHandler) CreateCarpoolRide(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000") 
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization") 

	if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
	}

	vars := mux.Vars(r)
	carpoolIDStr := vars["id"] 

	carpoolID, err := uuid.Parse(carpoolIDStr)
	if err != nil {
			log.Printf("Invalid carpool ID: %v", err)
			http.Error(w, "Invalid carpool ID", http.StatusBadRequest)
			return
	}

	log.Printf("Creating carpool ride for carpoolID: %s", carpoolID) 

	var ride models.CarpoolRide
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&ride)
	if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
	}

	// **Assign carpoolID from URL**
	ride.CarpoolID = carpoolID 

	ctx := context.Background()
	err = h.carpoolRideRepo.CreateCarpoolRide(ctx, &ride)
	if err != nil {
			log.Printf("failed to create carpool ride: %v\n", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ride)
}

func (h *CarPoolRideHandler) GetCarpoolRide(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	rideIDStr := vars["rideID"]

	rideID, err := uuid.Parse(rideIDStr)
	if err != nil {
			http.Error(w, "Invalid ride ID", http.StatusBadRequest)
			return
	}

	ride, err := h.carpoolRideRepo.GetCarpoolRide(r.Context(), rideID)
	if err != nil {
			if err == sql.ErrNoRows {
					http.Error(w, "Carpool ride not found", http.StatusNotFound)
					return
			}
			http.Error(w, fmt.Sprintf("Failed to get carpool ride: %v", err), http.StatusInternalServerError)
			return
	}

	json.NewEncoder(w).Encode(ride)
}

func (h *CarPoolRideHandler) DeleteCarpoolRide(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request: Method: %s, URL: %s", r.Method, r.URL) 

	vars := mux.Vars(r)
	carpoolIDStr := vars["carpoolID"] 
	rideIDStr := vars["rideID"] 

	carpoolID, err := uuid.Parse(carpoolIDStr)
	if err != nil {
			http.Error(w, "Invalid carpool ID", http.StatusBadRequest)
			return
	}

	rideID, err := uuid.Parse(rideIDStr)
	if err != nil {
			http.Error(w, "Invalid ride ID", http.StatusBadRequest)
			return
	}

	err = h.carpoolRideRepo.DeleteCarpoolRide(r.Context(), carpoolID, rideID) 
	if err != nil {
			if err.Error() == "carpool ride not found" {
					http.Error(w, "Carpool ride not found", http.StatusNotFound)
					return
			}
			log.Printf("Failed to delete carpool ride: %v", err)
			http.Error(w, "Failed to delete carpool ride", http.StatusInternalServerError)
			return
	}

	w.WriteHeader(http.StatusNoContent) 
}