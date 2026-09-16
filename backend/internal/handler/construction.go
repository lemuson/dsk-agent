package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"backend/internal/domain"
	"backend/internal/service"

	"github.com/go-chi/chi/v5"
)

type ConstructionHandler struct {
	service service.ConstructionService
}

func NewConstructionHandler(s service.ConstructionService) *ConstructionHandler {
	return &ConstructionHandler{service: s}
}

func parseID(r *http.Request) (int, error) {
	return strconv.Atoi(chi.URLParam(r, "id"))
}

func (h *ConstructionHandler) CreateComplex(w http.ResponseWriter, r *http.Request) {
	var complex domain.ResidentialComplex
	if err := json.NewDecoder(r.Body).Decode(&complex); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if complex.Name == "" || complex.Address == "" {
		http.Error(w, "name and address are required", http.StatusBadRequest)
		return
	}
	if err := h.service.CreateResidentialComplex(r.Context(), &complex); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(complex)
}

func (h *ConstructionHandler) GetComplex(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	complex, err := h.service.GetResidentialComplexByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(complex)
}

func (h *ConstructionHandler) GetAllComplexes(w http.ResponseWriter, r *http.Request) {
	complexes, err := h.service.GetAllResidentialComplexes(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if complexes == nil {
		complexes = []*domain.ResidentialComplex{}
	}
	json.NewEncoder(w).Encode(complexes)
}

func (h *ConstructionHandler) UpdateComplex(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var complex domain.ResidentialComplex
	if err := json.NewDecoder(r.Body).Decode(&complex); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if complex.Name == "" || complex.Address == "" {
		http.Error(w, "name and address are required", http.StatusBadRequest)
		return
	}
	complex.ID = id
	if err := h.service.UpdateResidentialComplex(r.Context(), &complex); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(complex)
}

func (h *ConstructionHandler) DeleteComplex(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.service.DeleteResidentialComplex(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ConstructionHandler) CreateBuilding(w http.ResponseWriter, r *http.Request) {
	var building domain.Building
	if err := json.NewDecoder(r.Body).Decode(&building); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if building.ResidentialComplexID == 0 || building.Address == "" || building.Status == "" || building.TypeWallMaterial == "" {
		http.Error(w, "residential_complex_id, address, status, and type_wall_material are required", http.StatusBadRequest)
		return
	}
	if err := h.service.CreateBuilding(r.Context(), &building); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(building)
}

func (h *ConstructionHandler) GetBuilding(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	building, err := h.service.GetBuildingByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(building)
}

func (h *ConstructionHandler) GetBuildingsByComplex(w http.ResponseWriter, r *http.Request) {
	complexID, err := strconv.Atoi(chi.URLParam(r, "complexId"))
	if err != nil {
		http.Error(w, "invalid complexId", http.StatusBadRequest)
		return
	}
	buildings, err := h.service.GetBuildingsByComplexID(r.Context(), complexID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if buildings == nil {
		buildings = []*domain.Building{}
	}
	json.NewEncoder(w).Encode(buildings)
}

func (h *ConstructionHandler) UpdateBuilding(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var building domain.Building
	if err := json.NewDecoder(r.Body).Decode(&building); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if building.ResidentialComplexID == 0 || building.Address == "" || building.Status == "" || building.TypeWallMaterial == "" {
		http.Error(w, "residential_complex_id, address, status, and type_wall_material are required", http.StatusBadRequest)
		return
	}
	building.ID = id
	if err := h.service.UpdateBuilding(r.Context(), &building); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(building)
}

func (h *ConstructionHandler) DeleteBuilding(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.service.DeleteBuilding(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ConstructionHandler) CreateApartment(w http.ResponseWriter, r *http.Request) {
	var apartment domain.Apartment
	if err := json.NewDecoder(r.Body).Decode(&apartment); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if apartment.BuildingID == 0 || apartment.Number == "" || apartment.Rooms == 0 || apartment.Area == 0 || apartment.Price == 0 || apartment.Status == "" {
		http.Error(w, "building_id, number, rooms, area, price, and status are required", http.StatusBadRequest)
		return
	}
	if err := h.service.CreateApartment(r.Context(), &apartment); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(apartment)
}

func (h *ConstructionHandler) GetApartment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	apartment, err := h.service.GetApartmentByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(apartment)
}

func (h *ConstructionHandler) GetApartmentsByBuilding(w http.ResponseWriter, r *http.Request) {
	buildingID, err := strconv.Atoi(chi.URLParam(r, "buildingId"))
	if err != nil {
		http.Error(w, "invalid buildingId", http.StatusBadRequest)
		return
	}
	apartments, err := h.service.GetApartmentsByBuildingID(r.Context(), buildingID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if apartments == nil {
		apartments = []*domain.Apartment{}
	}
	json.NewEncoder(w).Encode(apartments)
}

func (h *ConstructionHandler) UpdateApartment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var apartment domain.Apartment
	if err := json.NewDecoder(r.Body).Decode(&apartment); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if apartment.BuildingID == 0 || apartment.Number == "" || apartment.Rooms == 0 || apartment.Area == 0 || apartment.Price == 0 || apartment.Status == "" {
		http.Error(w, "building_id, number, rooms, area, price, and status are required", http.StatusBadRequest)
		return
	}
	apartment.ID = id
	if err := h.service.UpdateApartment(r.Context(), &apartment); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(apartment)
}

func (h *ConstructionHandler) DeleteApartment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.service.DeleteApartment(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ConstructionHandler) CreateProgress(w http.ResponseWriter, r *http.Request) {
	var progress domain.ConstructionProgress
	if err := json.NewDecoder(r.Body).Decode(&progress); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if progress.BuildingID == 0 || progress.StageName == "" || progress.Status == "" {
		http.Error(w, "building_id, stage_name, and status are required", http.StatusBadRequest)
		return
	}
	if err := h.service.CreateProgress(r.Context(), &progress); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(progress)
}

func (h *ConstructionHandler) GetProgress(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	progress, err := h.service.GetProgressByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(progress)
}

func (h *ConstructionHandler) GetProgressByBuilding(w http.ResponseWriter, r *http.Request) {
	buildingID, err := strconv.Atoi(chi.URLParam(r, "buildingId"))
	if err != nil {
		http.Error(w, "invalid buildingId", http.StatusBadRequest)
		return
	}
	progress, err := h.service.GetProgressByBuildingID(r.Context(), buildingID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if progress == nil {
		progress = []*domain.ConstructionProgress{}
	}
	json.NewEncoder(w).Encode(progress)
}

func (h *ConstructionHandler) UpdateProgress(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var progress domain.ConstructionProgress
	if err := json.NewDecoder(r.Body).Decode(&progress); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if progress.BuildingID == 0 || progress.StageName == "" || progress.Status == "" {
		http.Error(w, "building_id, stage_name, and status are required", http.StatusBadRequest)
		return
	}
	progress.ID = id
	if err := h.service.UpdateProgress(r.Context(), &progress); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(progress)
}

func (h *ConstructionHandler) DeleteProgress(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.service.DeleteProgress(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
