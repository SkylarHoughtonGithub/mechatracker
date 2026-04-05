package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/mechatracker/api/internal/apierr"
	"github.com/mechatracker/api/internal/middleware"
	"github.com/mechatracker/api/internal/store"
)

// VehicleHandler holds the sqlc Querier. Using the interface keeps
// this testable — inject store.MockQuerier in unit tests.
type VehicleHandler struct {
	q   store.Querier
	log *zap.Logger
}

func NewVehicleHandler(q store.Querier, log *zap.Logger) *VehicleHandler {
	return &VehicleHandler{q: q, log: log}
}

// ----------------------------------------------------------------
// GET /v1/vehicles
// Lists all vehicles belonging to the caller's org.
// ----------------------------------------------------------------

func (h *VehicleHandler) List(c *gin.Context) {
	orgID, ok := orgIDFromCtx(c)
	if !ok {
		return
	}

	vehicles, err := h.q.ListVehicles(c.Request.Context(), orgID)
	if err != nil {
		h.log.Error("ListVehicles", zap.Error(err), zap.String("org_id", orgID.String()))
		apierr.Internal(c)
		return
	}

	c.JSON(http.StatusOK, vehicles)
}

// ----------------------------------------------------------------
// GET /v1/vehicles/:vehicleID
// ----------------------------------------------------------------

func (h *VehicleHandler) Get(c *gin.Context) {
	orgID, ok := orgIDFromCtx(c)
	if !ok {
		return
	}
	vehicleID, ok := parseUUIDParam(c, "vehicleID")
	if !ok {
		return
	}

	vehicle, err := h.q.GetVehicle(c.Request.Context(), store.GetVehicleParams{
		ID:    vehicleID,
		OrgID: orgID,
	})
	if err != nil {
		apierr.NotFound(c, "Vehicle not found.")
		return
	}

	c.JSON(http.StatusOK, vehicle)
}

// ----------------------------------------------------------------
// POST /v1/vehicles
// ----------------------------------------------------------------

type createVehicleReq struct {
	Year      int16   `json:"year"       binding:"required,min=1900,max=2100"`
	Make      string  `json:"make"       binding:"required,max=100"`
	Model     string  `json:"model"      binding:"required,max=100"`
	Trim      *string `json:"trim"`
	VIN       *string `json:"vin"        binding:"omitempty,len=17"`
	Mileage   *int32  `json:"mileage"    binding:"omitempty,min=0"`
	UnitLabel *string `json:"unit_label" binding:"omitempty,max=50"`
}

func (h *VehicleHandler) Create(c *gin.Context) {
	orgID, ok := orgIDFromCtx(c)
	if !ok {
		return
	}

	var req createVehicleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(c, err.Error())
		return
	}

	vehicle, err := h.q.CreateVehicle(c.Request.Context(), store.CreateVehicleParams{
		OrgID:     orgID,
		Year:      req.Year,
		Make:      req.Make,
		Model:     req.Model,
		Trim:      req.Trim,
		Vin:       req.VIN,
		Mileage:   req.Mileage,
		UnitLabel: req.UnitLabel,
	})
	if err != nil {
		h.log.Error("CreateVehicle", zap.Error(err))
		apierr.Internal(c)
		return
	}

	c.JSON(http.StatusCreated, vehicle)
}

// ----------------------------------------------------------------
// PATCH /v1/vehicles/:vehicleID
// All fields optional — only provided fields are updated (sqlc narg).
// ----------------------------------------------------------------

type updateVehicleReq struct {
	Year      *int16  `json:"year"       binding:"omitempty,min=1900,max=2100"`
	Make      *string `json:"make"       binding:"omitempty,max=100"`
	Model     *string `json:"model"      binding:"omitempty,max=100"`
	Trim      *string `json:"trim"`
	VIN       *string `json:"vin"        binding:"omitempty,len=17"`
	Mileage   *int32  `json:"mileage"    binding:"omitempty,min=0"`
	UnitLabel *string `json:"unit_label" binding:"omitempty,max=50"`
}

func (h *VehicleHandler) Update(c *gin.Context) {
	orgID, ok := orgIDFromCtx(c)
	if !ok {
		return
	}
	vehicleID, ok := parseUUIDParam(c, "vehicleID")
	if !ok {
		return
	}

	var req updateVehicleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(c, err.Error())
		return
	}

	vehicle, err := h.q.UpdateVehicle(c.Request.Context(), store.UpdateVehicleParams{
		ID:        vehicleID,
		OrgID:     orgID,
		Year:      req.Year,
		Make:      req.Make,
		Model:     req.Model,
		Trim:      req.Trim,
		Vin:       req.VIN,
		Mileage:   req.Mileage,
		UnitLabel: req.UnitLabel,
	})
	if err != nil {
		apierr.NotFound(c, "Vehicle not found.")
		return
	}

	c.JSON(http.StatusOK, vehicle)
}

// ----------------------------------------------------------------
// DELETE /v1/vehicles/:vehicleID
// Hard deletes the vehicle and writes a snapshot to deleted_items
// so it can be recovered within 30 days.
// ----------------------------------------------------------------

func (h *VehicleHandler) Delete(c *gin.Context) {
	orgID, ok := orgIDFromCtx(c)
	if !ok {
		return
	}
	vehicleID, ok := parseUUIDParam(c, "vehicleID")
	if !ok {
		return
	}
	callerID, ok := callerIDFromCtx(c)
	if !ok {
		return
	}

	// DeleteVehicle returns the deleted row — we snapshot it before it's gone.
	deleted, err := h.q.DeleteVehicle(c.Request.Context(), store.DeleteVehicleParams{
		ID:    vehicleID,
		OrgID: orgID,
	})
	if err != nil {
		apierr.NotFound(c, "Vehicle not found.")
		return
	}

	// Write to deleted_items buffer (best-effort — log failure but don't 500).
	snapshot, _ := json.Marshal(deleted)
	if _, err := h.q.InsertDeletedItem(c.Request.Context(), store.InsertDeletedItemParams{
		EntityType: "vehicle",
		EntityID:   deleted.ID,
		OrgID:      orgID,
		DeletedBy:  callerID,
		Snapshot:   snapshot,
	}); err != nil {
		h.log.Error("InsertDeletedItem after vehicle delete",
			zap.Error(err),
			zap.String("vehicle_id", vehicleID.String()),
		)
	}

	c.JSON(http.StatusOK, gin.H{"deleted": true, "id": vehicleID})
}

// ----------------------------------------------------------------
// helpers — shared by all handlers in this package
// ----------------------------------------------------------------

// orgIDFromCtx reads the Clerk org_id set by middleware.Auth and
// parses it as a UUID. Aborts with 401 if missing or malformed.
func orgIDFromCtx(c *gin.Context) (uuid.UUID, bool) {
	raw := c.GetString(middleware.KeyOrgID)
	if raw == "" {
		apierr.Unauthorized(c, "No organization context in token.")
		return uuid.UUID{}, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		apierr.Unauthorized(c, "Malformed organization ID.")
		return uuid.UUID{}, false
	}
	return id, true
}

// callerIDFromCtx reads the internal users.id from context.
//
// In dev bypass mode, KeyUserID is set to DevUserID (a real UUID) so this
// parses cleanly. When real Clerk auth is wired in, KeyUserID will be the
// Clerk subject string ("user_2abc...") which is not a UUID — at that point
// add a user-lookup middleware that queries GetUserByClerkID and stores the
// internal UUID under a separate key (e.g. KeyInternalUserID), then read
// that key here instead.
func callerIDFromCtx(c *gin.Context) (uuid.UUID, bool) {
	raw := c.GetString(middleware.KeyUserID)
	if raw == "" {
		apierr.Unauthorized(c, "No User context.")
		return uuid.UUID{}, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		apierr.Unauthorized(c, "Malformed user ID.")
		return uuid.UUID{}, false
	}
	return id, true
}

// parseUUIDParam parses a Gin URL parameter as a UUID.
// Aborts with 400 if the value is not a valid UUID.
func parseUUIDParam(c *gin.Context, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(param))
	if err != nil {
		apierr.BadRequest(c, "Invalid "+param+": must be a UUID.")
		return uuid.UUID{}, false
	}
	return id, true
}
