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

// List godoc
// @Summary      List vehicles
// @Description  Returns all vehicles belonging to the caller's org.
// @Tags         vehicles
// @Produce      json
// @Success      200  {array}   SwaggerVehicle
// @Failure      401  {object}  apierr.ErrorResponse
// @Failure      500  {object}  apierr.ErrorResponse
// @Security     BearerAuth
// @Router       /vehicles [get]
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

// Get godoc
// @Summary      Get a vehicle
// @Description  Returns a single vehicle by ID, scoped to the caller's org.
// @Tags         vehicles
// @Produce      json
// @Param        vehicleID  path      string  true  "Vehicle UUID"  format(uuid)
// @Success      200        {object}  SwaggerVehicle
// @Failure      400        {object}  apierr.ErrorResponse
// @Failure      401        {object}  apierr.ErrorResponse
// @Failure      404        {object}  apierr.ErrorResponse
// @Security     BearerAuth
// @Router       /vehicles/{vehicleID} [get]
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

// createVehicleReq is the request body for creating a vehicle.
type createVehicleReq struct {
	Year      int16   `json:"year"       binding:"required,min=1900,max=2100" example:"2019"`
	Make      string  `json:"make"       binding:"required,max=100"           example:"Ford"`
	Model     string  `json:"model"      binding:"required,max=100"           example:"F-150"`
	Trim      *string `json:"trim"                                            example:"XLT"`
	VIN       *string `json:"vin"        binding:"omitempty,len=17"           example:"1FTFW1ET5DFC10312"`
	Mileage   *int32  `json:"mileage"    binding:"omitempty,min=0"            example:"84200"`
	UnitLabel *string `json:"unit_label" binding:"omitempty,max=50"           example:"Unit #07"`
}

// Create godoc
// @Summary      Create a vehicle
// @Description  Adds a vehicle to the caller's org garage.
// @Tags         vehicles
// @Accept       json
// @Produce      json
// @Param        body  body      createVehicleReq  true  "Vehicle details"
// @Success      201   {object}  SwaggerVehicle
// @Failure      400   {object}  apierr.ErrorResponse
// @Failure      401   {object}  apierr.ErrorResponse
// @Failure      500   {object}  apierr.ErrorResponse
// @Security     BearerAuth
// @Router       /vehicles [post]
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

// updateVehicleReq is the request body for updating a vehicle.
// All fields are optional — only provided fields are updated.
type updateVehicleReq struct {
	Year      *int16  `json:"year"       binding:"omitempty,min=1900,max=2100" example:"2020"`
	Make      *string `json:"make"       binding:"omitempty,max=100"           example:"Ford"`
	Model     *string `json:"model"      binding:"omitempty,max=100"           example:"F-150"`
	Trim      *string `json:"trim"                                            example:"Lariat"`
	VIN       *string `json:"vin"        binding:"omitempty,len=17"`
	Mileage   *int32  `json:"mileage"    binding:"omitempty,min=0"            example:"92000"`
	UnitLabel *string `json:"unit_label" binding:"omitempty,max=50"`
}

// Update godoc
// @Summary      Update a vehicle
// @Description  Partially updates a vehicle. Only provided fields are changed.
// @Tags         vehicles
// @Accept       json
// @Produce      json
// @Param        vehicleID  path      string            true  "Vehicle UUID"  format(uuid)
// @Param        body       body      updateVehicleReq  true  "Fields to update"
// @Success      200        {object}  SwaggerVehicle
// @Failure      400        {object}  apierr.ErrorResponse
// @Failure      401        {object}  apierr.ErrorResponse
// @Failure      404        {object}  apierr.ErrorResponse
// @Security     BearerAuth
// @Router       /vehicles/{vehicleID} [patch]
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

// Delete godoc
// @Summary      Delete a vehicle
// @Description  Hard-deletes a vehicle and writes a 30-day recoverable snapshot to deleted_items.
// @Tags         vehicles
// @Produce      json
// @Param        vehicleID  path      string  true  "Vehicle UUID"  format(uuid)
// @Success      200        {object}  SwaggerDeletedResponse  "deleted: true, id: <uuid>"
// @Failure      400        {object}  apierr.ErrorResponse
// @Failure      401        {object}  apierr.ErrorResponse
// @Failure      404        {object}  apierr.ErrorResponse
// @Security     BearerAuth
// @Router       /vehicles/{vehicleID} [delete]
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

	deleted, err := h.q.DeleteVehicle(c.Request.Context(), store.DeleteVehicleParams{
		ID:    vehicleID,
		OrgID: orgID,
	})
	if err != nil {
		apierr.NotFound(c, "Vehicle not found.")
		return
	}

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

// DeleteSystem godoc
// @Summary      Delete all repairs in a system for a vehicle
// @Description  Deletes all repairs (and their steps, parts, notes) for the given vehicle+system.
// @Tags         vehicles
// @Produce      json
// @Param        vehicleID  path      string  true  "Vehicle UUID"  format(uuid)
// @Param        systemKey  path      string  true  "System key (e.g. winch)"
// @Success      200        {object}  map[string]interface{}
// @Failure      400        {object}  apierr.ErrorResponse
// @Failure      401        {object}  apierr.ErrorResponse
// @Security     BearerAuth
// @Router       /vehicles/{vehicleID}/systems/{systemKey} [delete]
func (h *VehicleHandler) DeleteSystem(c *gin.Context) {
	orgID, ok := orgIDFromCtx(c)
	if !ok {
		return
	}
	vehicleID, ok := parseUUIDParam(c, "vehicleID")
	if !ok {
		return
	}
	systemKey := c.Param("systemKey")
	if systemKey == "" {
		apierr.BadRequest(c, "systemKey is required.")
		return
	}

	if err := h.q.DeleteVehicleSystem(c.Request.Context(), store.DeleteVehicleSystemParams{
		VehicleID: vehicleID,
		OrgID:     orgID,
		System:    systemKey,
	}); err != nil {
		h.log.Error("DeleteVehicleSystem", zap.Error(err), zap.String("system", systemKey))
		apierr.Internal(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": true, "system": systemKey})
}

// ----------------------------------------------------------------
// helpers — shared by all handlers in this package
// ----------------------------------------------------------------

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

// callerIDFromCtx reads the internal users.id from context. RequireUser (or
// Auth with a post-onboarding session) must have run first.
func callerIDFromCtx(c *gin.Context) (uuid.UUID, bool) {
	raw := c.GetString(middleware.KeyUserID)
	if raw == "" {
		apierr.Unauthorized(c, "No user context.")
		return uuid.UUID{}, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		apierr.Unauthorized(c, "Malformed user ID.")
		return uuid.UUID{}, false
	}
	return id, true
}

func parseUUIDParam(c *gin.Context, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(param))
	if err != nil {
		apierr.BadRequest(c, "Invalid "+param+": must be a UUID.")
		return uuid.UUID{}, false
	}
	return id, true
}
