package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/mechatracker/api/internal/apierr"
	"github.com/mechatracker/api/internal/store"
)

// RepairHandler handles all repair-related endpoints.
type RepairHandler struct {
	q   store.Querier
	log *zap.Logger
}

func NewRepairHandler(q store.Querier, log *zap.Logger) *RepairHandler {
	return &RepairHandler{q: q, log: log}
}

// ── Response types ────────────────────────────────────────────────────────────

// repairDetail is the rich GET /:repairID response — repair + steps + parts in one call.
type repairDetail struct {
	store.Repair
	Steps []store.RepairStep `json:"steps"`
	Parts []store.Part       `json:"parts"`
}

// List godoc
// @Summary      List active repairs
// @Description  Returns in-progress repairs assigned to or created by the caller.
// @Tags         repairs
// @Produce      json
// @Success      200  {array}   SwaggerActiveRepair
// @Failure      401  {object}  apierr.ErrorResponse
// @Failure      500  {object}  apierr.ErrorResponse
// @Security     BearerAuth
// @Router       /repairs [get]
func (h *RepairHandler) List(c *gin.Context) {
	orgID, ok := orgIDFromCtx(c)
	if !ok {
		return
	}
	callerID, ok := callerIDFromCtx(c)
	if !ok {
		return
	}

	repairs, err := h.q.ListActiveRepairs(c.Request.Context(), store.ListActiveRepairsParams{
		OrgID:      orgID,
		AssignedTo: uuidToPgtype(callerID),
	})
	if err != nil {
		h.log.Error("ListActiveRepairs", zap.Error(err))
		apierr.Internal(c)
		return
	}

	c.JSON(http.StatusOK, repairs)
}

// createRepairReq is the request body for creating a repair.
type createRepairReq struct {
	VehicleID  string  `json:"vehicle_id"  binding:"required,uuid" example:"a44d8522-66b3-4146-af64-9db60c72470a"`
	Title      string  `json:"title"       binding:"required,max=200" example:"Brake pad replacement"`
	System     string  `json:"system"      binding:"required,max=100" example:"brakes"`
	AssignedTo *string `json:"assigned_to" binding:"omitempty,uuid"`
	TemplateID *string `json:"template_id" binding:"omitempty,uuid"`
}

// Create godoc
// @Summary      Create a repair
// @Description  Creates a new repair in draft status for a vehicle.
// @Tags         repairs
// @Accept       json
// @Produce      json
// @Param        body  body      createRepairReq  true  "Repair details"
// @Success      201   {object}  SwaggerRepair
// @Failure      400   {object}  apierr.ErrorResponse
// @Failure      401   {object}  apierr.ErrorResponse
// @Failure      500   {object}  apierr.ErrorResponse
// @Security     BearerAuth
// @Router       /repairs [post]
func (h *RepairHandler) Create(c *gin.Context) {
	orgID, ok := orgIDFromCtx(c)
	if !ok {
		return
	}
	callerID, ok := callerIDFromCtx(c)
	if !ok {
		return
	}

	var req createRepairReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(c, err.Error())
		return
	}

	vehicleID, err := uuid.Parse(req.VehicleID)
	if err != nil {
		apierr.BadRequest(c, "Invalid vehicle_id: must be a UUID.")
		return
	}

	params := store.CreateRepairParams{
		VehicleID: vehicleID,
		OrgID:     orgID,
		CreatedBy: callerID,
		Title:     req.Title,
		System:    req.System,
	}

	if req.AssignedTo != nil {
		id, err := uuid.Parse(*req.AssignedTo)
		if err != nil {
			apierr.BadRequest(c, "Invalid assigned_to: must be a UUID.")
			return
		}
		params.AssignedTo = uuidToPgtype(id)
	}

	if req.TemplateID != nil {
		id, err := uuid.Parse(*req.TemplateID)
		if err != nil {
			apierr.BadRequest(c, "Invalid template_id: must be a UUID.")
			return
		}
		params.TemplateID = uuidToPgtype(id)
	}

	repair, err := h.q.CreateRepair(c.Request.Context(), params)
	if err != nil {
		h.log.Error("CreateRepair", zap.Error(err))
		apierr.Internal(c)
		return
	}

	c.JSON(http.StatusCreated, repair)
}

// Get godoc
// @Summary      Get a repair
// @Description  Returns the repair record with its steps and parts in one response.
// @Tags         repairs
// @Produce      json
// @Param        repairID  path      string  true  "Repair UUID"  format(uuid)
// @Success      200       {object}  SwaggerRepairDetail
// @Failure      400       {object}  apierr.ErrorResponse
// @Failure      401       {object}  apierr.ErrorResponse
// @Failure      404       {object}  apierr.ErrorResponse
// @Failure      500       {object}  apierr.ErrorResponse
// @Security     BearerAuth
// @Router       /repairs/{repairID} [get]
func (h *RepairHandler) Get(c *gin.Context) {
	orgID, ok := orgIDFromCtx(c)
	if !ok {
		return
	}
	repairID, ok := parseUUIDParam(c, "repairID")
	if !ok {
		return
	}

	repair, err := h.q.GetRepair(c.Request.Context(), store.GetRepairParams{
		ID:    repairID,
		OrgID: orgID,
	})
	if err != nil {
		apierr.NotFound(c, "Repair not found.")
		return
	}

	var steps []store.RepairStep
	var parts []store.Part

	g, ctx := errgroup.WithContext(c.Request.Context())
	g.Go(func() error {
		var err error
		steps, err = h.q.ListSteps(ctx, repairID)
		return err
	})
	g.Go(func() error {
		var err error
		parts, err = h.q.ListParts(ctx, repairID)
		return err
	})

	if err := g.Wait(); err != nil {
		h.log.Error("GetRepair detail", zap.Error(err), zap.String("repair_id", repairID.String()))
		apierr.Internal(c)
		return
	}

	c.JSON(http.StatusOK, repairDetail{
		Repair: repair,
		Steps:  steps,
		Parts:  parts,
	})
}

// updateRepairReq is the request body for updating a repair.
// Provide `status` for a status transition, or field edits otherwise — not both.
type updateRepairReq struct {
	// Status transition — mutually exclusive with field edits.
	Status *string `json:"status" binding:"omitempty,oneof=draft in_progress pending_parts completed" example:"in_progress"`

	// Field edits — all optional.
	Title      *string `json:"title"       binding:"omitempty,max=200" example:"Brake pad replacement"`
	System     *string `json:"system"      binding:"omitempty,max=100" example:"brakes"`
	AssignedTo *string `json:"assigned_to" binding:"omitempty,uuid"`
}

// Add to api/internal/handler/vehicle.go
// (Insert after the existing Delete handler, before the helper functions section)

// ListRepairs godoc
// @Summary      List repairs for a vehicle
// @Description  Returns all repairs for a vehicle, optionally filtered by system.
// @Tags         vehicles
// @Produce      json
// @Param        vehicleID  path      string  true   "Vehicle UUID"  format(uuid)
// @Param        system     query     string  false  "Filter by system key (e.g. engine, brakes)"
// @Success      200        {array}   SwaggerRepair
// @Failure      400        {object}  apierr.ErrorResponse
// @Failure      401        {object}  apierr.ErrorResponse
// @Failure      404        {object}  apierr.ErrorResponse
// @Failure      500        {object}  apierr.ErrorResponse
// @Security     BearerAuth
// @Router       /vehicles/{vehicleID}/repairs [get]
func (h *VehicleHandler) ListRepairs(c *gin.Context) {
	orgID, ok := orgIDFromCtx(c)
	if !ok {
		return
	}
	vehicleID, ok := parseUUIDParam(c, "vehicleID")
	if !ok {
		return
	}

	// Verify the vehicle exists and belongs to this org
	if _, err := h.q.GetVehicle(c.Request.Context(), store.GetVehicleParams{
		ID:    vehicleID,
		OrgID: orgID,
	}); err != nil {
		apierr.NotFound(c, "Vehicle not found.")
		return
	}

	repairs, err := h.q.ListRepairsByVehicle(c.Request.Context(), store.ListRepairsByVehicleParams{
		VehicleID: vehicleID,
		OrgID:     orgID,
	})
	if err != nil {
		h.log.Error("ListRepairsByVehicle", zap.Error(err), zap.String("vehicle_id", vehicleID.String()))
		apierr.Internal(c)
		return
	}

	// Filter by system query param if provided
	system := c.Query("system")
	if system != "" {
		filtered := repairs[:0]
		for _, r := range repairs {
			if r.System == system {
				filtered = append(filtered, r)
			}
		}
		repairs = filtered
	}

	c.JSON(http.StatusOK, repairs)
}

// Update godoc
// @Summary      Update a repair
// @Description  Handles two operations: status transition (provide `status`) or field edits (provide `title`/`system`/`assigned_to`). Completing a repair writes an immutable history snapshot.
// @Tags         repairs
// @Accept       json
// @Produce      json
// @Param        repairID  path      string           true  "Repair UUID"  format(uuid)
// @Param        body      body      updateRepairReq  true  "Update payload"
// @Success      200       {object}  SwaggerRepair
// @Failure      400       {object}  apierr.ErrorResponse
// @Failure      401       {object}  apierr.ErrorResponse
// @Failure      404       {object}  apierr.ErrorResponse
// @Security     BearerAuth
// @Router       /repairs/{repairID} [patch]
func (h *RepairHandler) Update(c *gin.Context) {
	orgID, ok := orgIDFromCtx(c)
	if !ok {
		return
	}
	repairID, ok := parseUUIDParam(c, "repairID")
	if !ok {
		return
	}

	var req updateRepairReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(c, err.Error())
		return
	}

	if req.Status != nil {
		current, err := h.q.GetRepair(c.Request.Context(), store.GetRepairParams{
			ID:    repairID,
			OrgID: orgID,
		})
		if err != nil {
			apierr.NotFound(c, "Repair not found.")
			return
		}

		to := store.RepairStatus(*req.Status)
		if !isValidTransition(current.Status, to) {
			apierr.BadRequest(c, fmt.Sprintf(
				"Cannot transition repair from %q to %q.", current.Status, to,
			))
			return
		}

		repair, err := h.q.UpdateRepairStatus(c.Request.Context(), store.UpdateRepairStatusParams{
			ID:     repairID,
			OrgID:  orgID,
			Status: to,
		})
		if err != nil {
			apierr.NotFound(c, "Repair not found.")
			return
		}

		if repair.Status == store.RepairStatusCompleted {
			if err := h.writeRepairHistory(c, repair, orgID); err != nil {
				h.log.Error("writeRepairHistory failed",
					zap.Error(err),
					zap.String("repair_id", repair.ID.String()),
				)
			}
		}

		c.JSON(http.StatusOK, repair)
		return
	}

	params := store.UpdateRepairParams{
		ID:     repairID,
		OrgID:  orgID,
		Title:  req.Title,
		System: req.System,
	}

	if req.AssignedTo != nil {
		id, err := uuid.Parse(*req.AssignedTo)
		if err != nil {
			apierr.BadRequest(c, "Invalid assigned_to: must be a UUID.")
			return
		}
		params.AssignedTo = uuidToPgtype(id)
	}

	repair, err := h.q.UpdateRepair(c.Request.Context(), params)
	if err != nil {
		apierr.NotFound(c, "Repair not found.")
		return
	}

	c.JSON(http.StatusOK, repair)
}

// Delete godoc
// @Summary     Delete a repair
// @Description Hard-deletes a repair and writes a 30-day recoverable snapshot to deleted_items.
// @Tags        repairs
// @Produce     json
// @Param       repairID path     string  true  "Repair UUID"  format(uuid)
// @Success     200      {object} SwaggerDeletedResponse
// @Failure     401      {object} apierr.ErrorResponse
// @Failure     404      {object} apierr.ErrorResponse
// @Security    BearerAuth
// @Router      /repairs/{repairID} [delete]
func (h *RepairHandler) Delete(c *gin.Context) {
	orgID, ok := orgIDFromCtx(c)
	if !ok {
		return
	}
	repairID, ok := parseUUIDParam(c, "repairID")
	if !ok {
		return
	}
	callerID, ok := callerIDFromCtx(c)
	if !ok {
		return
	}

	repair, err := h.q.GetRepair(c.Request.Context(), store.GetRepairParams{
		ID:    repairID,
		OrgID: orgID,
	})
	if err != nil {
		apierr.NotFound(c, "Repair not found.")
		return
	}

	// Snapshot before delete
	snapshot, _ := json.Marshal(repair)
	if _, err := h.q.InsertDeletedItem(c.Request.Context(), store.InsertDeletedItemParams{
		EntityType: "repair",
		EntityID:   repairID,
		OrgID:      orgID,
		DeletedBy:  callerID,
		Snapshot:   snapshot,
	}); err != nil {
		h.log.Error("InsertDeletedItem before repair delete", zap.Error(err))
	}

	if err := h.q.DeleteRepair(c.Request.Context(), store.DeleteRepairParams{
		ID:    repairID,
		OrgID: orgID,
	}); err != nil {
		h.log.Error("DeleteRepair", zap.Error(err))
		apierr.Internal(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": true, "id": repairID})
}

func (h *RepairHandler) writeRepairHistory(c *gin.Context, repair store.Repair, orgID uuid.UUID) error {
	ctx := c.Request.Context()

	var (
		vehicle   store.Vehicle
		assignee  *store.User
		steps     []store.RepairStep
		parts     []store.Part
		notes     []store.ListNotesRow
		resources []store.Resource
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		vehicle, err = h.q.GetVehicle(gctx, store.GetVehicleParams{
			ID:    repair.VehicleID,
			OrgID: orgID,
		})
		return err
	})
	g.Go(func() error {
		var err error
		steps, err = h.q.ListSteps(gctx, repair.ID)
		return err
	})
	g.Go(func() error {
		var err error
		parts, err = h.q.ListParts(gctx, repair.ID)
		return err
	})
	g.Go(func() error {
		var err error
		notes, err = h.q.ListNotes(gctx, repair.ID)
		return err
	})
	g.Go(func() error {
		var err error
		resources, err = h.q.ListResources(gctx, repair.ID)
		return err
	})

	if repair.AssignedTo.Valid {
		g.Go(func() error {
			u, err := h.q.GetUserByID(gctx, repair.AssignedTo.Bytes)
			if err != nil {
				return err
			}
			assignee = &u
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("snapshot fetch: %w", err)
	}

	vehicleSnapshot := buildVehicleSnapshot(vehicle)
	var assignedSnapshot *string
	if assignee != nil {
		s := assignee.Name
		assignedSnapshot = &s
	}

	stepsJSON, err := json.Marshal(steps)
	if err != nil {
		return fmt.Errorf("marshal steps: %w", err)
	}
	partsJSON, err := json.Marshal(parts)
	if err != nil {
		return fmt.Errorf("marshal parts: %w", err)
	}
	notesJSON, err := json.Marshal(notes)
	if err != nil {
		return fmt.Errorf("marshal notes: %w", err)
	}
	resourcesJSON, err := json.Marshal(resources)
	if err != nil {
		return fmt.Errorf("marshal resources: %w", err)
	}

	_, err = h.q.WriteRepairHistory(ctx, store.WriteRepairHistoryParams{
		RepairID:          repair.ID,
		VehicleID:         repair.VehicleID,
		OrgID:             repair.OrgID,
		CreatedBy:         repair.CreatedBy,
		AssignedTo:        repair.AssignedTo,
		VehicleSnapshot:   vehicleSnapshot,
		AssignedSnapshot:  assignedSnapshot,
		Title:             repair.Title,
		System:            repair.System,
		TotalPartsCost:    repair.TotalPartsCost,
		TotalSteps:        int16(len(steps)),
		StepsSnapshot:     stepsJSON,
		PartsSnapshot:     partsJSON,
		NotesSnapshot:     notesJSON,
		ResourcesSnapshot: resourcesJSON,
		StartedAt:         repair.StartedAt,
		CompletedAt:       repair.CompletedAt,
	})
	return err
}

func buildVehicleSnapshot(v store.Vehicle) string {
	s := fmt.Sprintf("%d %s %s", v.Year, v.Make, v.Model)
	if v.Trim != nil && *v.Trim != "" {
		s += " " + *v.Trim
	}
	if v.UnitLabel != nil && *v.UnitLabel != "" {
		s += " — " + *v.UnitLabel
	}
	return s
}

// createStepReq is the request body for adding a step to a repair.
type createStepReq struct {
	Position   int16   `json:"position"    binding:"required,min=1" example:"1"`
	Title      string  `json:"title"       binding:"required,max=200" example:"Remove wheel"`
	Notes      *string `json:"notes"       example:"Use 21mm socket"`
	TorqueSpec *string `json:"torque_spec" example:"100 ft-lbs"`
}

// CreateStep godoc
// @Summary      Add a step to a repair
// @Description  Adds a step to a repair. Only allowed when repair status is draft or in_progress.
// @Tags         repairs
// @Accept       json
// @Produce      json
// @Param        repairID  path      string          true  "Repair UUID"  format(uuid)
// @Param        body      body      createStepReq   true  "Step details"
// @Success      201       {object}  SwaggerRepairStep
// @Failure      400       {object}  apierr.ErrorResponse
// @Failure      401       {object}  apierr.ErrorResponse
// @Failure      404       {object}  apierr.ErrorResponse
// @Failure      500       {object}  apierr.ErrorResponse
// @Security     BearerAuth
// @Router       /repairs/{repairID}/steps [post]
func (h *RepairHandler) CreateStep(c *gin.Context) {
	orgID, ok := orgIDFromCtx(c)
	if !ok {
		return
	}
	repairID, ok := parseUUIDParam(c, "repairID")
	if !ok {
		return
	}

	repair, err := h.q.GetRepair(c.Request.Context(), store.GetRepairParams{
		ID:    repairID,
		OrgID: orgID,
	})
	if err != nil {
		apierr.NotFound(c, "Repair not found.")
		return
	}

	if !repairIsEditable(repair.Status) {
		apierr.BadRequest(c, "Steps can only be added to repairs with status draft or in_progress.")
		return
	}

	var req createStepReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(c, err.Error())
		return
	}

	step, err := h.q.CreateStep(c.Request.Context(), store.CreateStepParams{
		RepairID:   repairID,
		Position:   req.Position,
		Title:      req.Title,
		Notes:      req.Notes,
		TorqueSpec: req.TorqueSpec,
	})
	if err != nil {
		h.log.Error("CreateStep", zap.Error(err))
		apierr.Internal(c)
		return
	}

	c.JSON(http.StatusCreated, step)
}

// DeleteStep godoc
// @Summary      Delete a step
// @Description  Removes a step from a repair. Only allowed when status is draft or in_progress.
// @Tags         repairs
// @Produce      json
// @Param        repairID  path      string  true  "Repair UUID"  format(uuid)
// @Param        stepID    path      string  true  "Step UUID"    format(uuid)
// @Success      200       {object}  SwaggerDeletedResponse  "deleted: true, id: <uuid>"
// @Failure      400       {object}  apierr.ErrorResponse
// @Failure      401       {object}  apierr.ErrorResponse
// @Failure      404       {object}  apierr.ErrorResponse
// @Security     BearerAuth
// @Router       /repairs/{repairID}/steps/{stepID} [delete]
func (h *RepairHandler) DeleteStep(c *gin.Context) {
	orgID, ok := orgIDFromCtx(c)
	if !ok {
		return
	}
	repairID, ok := parseUUIDParam(c, "repairID")
	if !ok {
		return
	}
	stepID, ok := parseUUIDParam(c, "stepID")
	if !ok {
		return
	}

	repair, err := h.q.GetRepair(c.Request.Context(), store.GetRepairParams{
		ID:    repairID,
		OrgID: orgID,
	})
	if err != nil {
		apierr.NotFound(c, "Repair not found.")
		return
	}

	if !repairIsEditable(repair.Status) {
		apierr.BadRequest(c, "Steps can only be removed from repairs with status draft or in_progress.")
		return
	}

	if err := h.q.DeleteStep(c.Request.Context(), store.DeleteStepParams{
		ID:       stepID,
		RepairID: repairID,
	}); err != nil {
		h.log.Error("DeleteStep", zap.Error(err))
		apierr.Internal(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": true, "id": stepID})
}

// CompleteStep godoc
// @Summary      Toggle step completion
// @Description  Toggles a step between complete and incomplete. Recalculates repair progress_pct after the toggle.
// @Tags         repairs
// @Produce      json
// @Param        repairID  path      string  true  "Repair UUID"  format(uuid)
// @Param        stepID    path      string  true  "Step UUID"    format(uuid)
// @Success      200       {object}  SwaggerStepToggleResponse  "step and repair objects"
// @Failure      400       {object}  apierr.ErrorResponse
// @Failure      401       {object}  apierr.ErrorResponse
// @Failure      404       {object}  apierr.ErrorResponse
// @Failure      500       {object}  apierr.ErrorResponse
// @Security     BearerAuth
// @Router       /repairs/{repairID}/steps/{stepID}/complete [post]
func (h *RepairHandler) CompleteStep(c *gin.Context) {
	orgID, ok := orgIDFromCtx(c)
	if !ok {
		return
	}
	repairID, ok := parseUUIDParam(c, "repairID")
	if !ok {
		return
	}
	stepID, ok := parseUUIDParam(c, "stepID")
	if !ok {
		return
	}

	if _, err := h.q.GetRepair(c.Request.Context(), store.GetRepairParams{
		ID:    repairID,
		OrgID: orgID,
	}); err != nil {
		apierr.NotFound(c, "Repair not found.")
		return
	}

	steps, err := h.q.ListSteps(c.Request.Context(), repairID)
	if err != nil {
		h.log.Error("ListSteps for toggle", zap.Error(err))
		apierr.Internal(c)
		return
	}

	var currentlyCompleted bool
	found := false
	for _, s := range steps {
		if s.ID == stepID {
			currentlyCompleted = s.Completed
			found = true
			break
		}
	}
	if !found {
		apierr.NotFound(c, "Step not found.")
		return
	}

	var updatedStep store.RepairStep
	if currentlyCompleted {
		updatedStep, err = h.q.UncompleteStep(c.Request.Context(), store.UncompleteStepParams{
			ID:       stepID,
			RepairID: repairID,
		})
	} else {
		updatedStep, err = h.q.CompleteStep(c.Request.Context(), store.CompleteStepParams{
			ID:       stepID,
			RepairID: repairID,
		})
	}
	if err != nil {
		h.log.Error("toggle step", zap.Error(err))
		apierr.Internal(c)
		return
	}

	counts, err := h.q.CountSteps(c.Request.Context(), repairID)
	if err != nil {
		h.log.Error("CountSteps", zap.Error(err))
		apierr.Internal(c)
		return
	}

	var pct int16
	if counts.Total > 0 {
		pct = int16(counts.Completed * 100 / counts.Total)
	}

	partsCost, err := h.q.SumPartsCost(c.Request.Context(), repairID)
	if err != nil {
		h.log.Error("SumPartsCost", zap.Error(err))
		apierr.Internal(c)
		return
	}

	updatedRepair, err := h.q.UpdateRepairProgress(c.Request.Context(), store.UpdateRepairProgressParams{
		ID:             repairID,
		ProgressPct:    pct,
		TotalPartsCost: partsCost,
	})
	if err != nil {
		h.log.Error("UpdateRepairProgress", zap.Error(err))
		apierr.Internal(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"step":   updatedStep,
		"repair": updatedRepair,
	})
}

// createNoteReq is the request body for adding a note.
type createNoteReq struct {
	Body string `json:"body" binding:"required,min=1,max=5000" example:"Noticed some scoring on the rotor — ordered replacement."`
}

// CreateNote godoc
// @Summary      Add a note to a repair
// @Description  Creates a timestamped note on the repair, attributed to the caller.
// @Tags         repairs
// @Accept       json
// @Produce      json
// @Param        repairID  path      string          true  "Repair UUID"  format(uuid)
// @Param        body      body      createNoteReq   true  "Note body"
// @Success      201       {object}  SwaggerRepairNote
// @Failure      400       {object}  apierr.ErrorResponse
// @Failure      401       {object}  apierr.ErrorResponse
// @Failure      404       {object}  apierr.ErrorResponse
// @Failure      500       {object}  apierr.ErrorResponse
// @Security     BearerAuth
// @Router       /repairs/{repairID}/notes [post]
func (h *RepairHandler) CreateNote(c *gin.Context) {
	orgID, ok := orgIDFromCtx(c)
	if !ok {
		return
	}
	repairID, ok := parseUUIDParam(c, "repairID")
	if !ok {
		return
	}
	callerID, ok := callerIDFromCtx(c)
	if !ok {
		return
	}

	if _, err := h.q.GetRepair(c.Request.Context(), store.GetRepairParams{
		ID:    repairID,
		OrgID: orgID,
	}); err != nil {
		apierr.NotFound(c, "Repair not found.")
		return
	}

	var req createNoteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(c, err.Error())
		return
	}

	note, err := h.q.CreateNote(c.Request.Context(), store.CreateNoteParams{
		RepairID: repairID,
		UserID:   callerID,
		Body:     req.Body,
	})
	if err != nil {
		h.log.Error("CreateNote", zap.Error(err))
		apierr.Internal(c)
		return
	}

	c.JSON(http.StatusCreated, note)
}

// UploadPhoto godoc
// @Summary      Get a photo upload URL
// @Description  Returns a presigned URL for direct-to-bucket photo upload. Not yet implemented — wire object storage config first.
// @Tags         repairs
// @Produce      json
// @Param        repairID  path      string  true  "Repair UUID"  format(uuid)
// @Success      200       {object}  map[string]interface{}  "upload_url"
// @Failure      400       {object}  apierr.ErrorResponse
// @Failure      401       {object}  apierr.ErrorResponse
// @Failure      404       {object}  apierr.ErrorResponse
// @Failure      501       {object}  map[string]interface{}  "not implemented"
// @Security     BearerAuth
// @Router       /repairs/{repairID}/photos [post]
func (h *RepairHandler) UploadPhoto(c *gin.Context) {
	orgID, ok := orgIDFromCtx(c)
	if !ok {
		return
	}
	repairID, ok := parseUUIDParam(c, "repairID")
	if !ok {
		return
	}

	if _, err := h.q.GetRepair(c.Request.Context(), store.GetRepairParams{
		ID:    repairID,
		OrgID: orgID,
	}); err != nil {
		apierr.NotFound(c, "Repair not found.")
		return
	}

	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "Photo upload not yet implemented — wire object storage in Phase 2.",
	})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// validTransitions defines the allowed status progressions for a repair.
// Any transition not listed here is rejected with a 400.
var validTransitions = map[store.RepairStatus][]store.RepairStatus{
	store.RepairStatusDraft:        {store.RepairStatusInProgress},
	store.RepairStatusInProgress:   {store.RepairStatusPendingParts, store.RepairStatusCompleted},
	store.RepairStatusPendingParts: {store.RepairStatusInProgress, store.RepairStatusCompleted},
	store.RepairStatusCompleted:    {store.RepairStatusInProgress},
}

func isValidTransition(from, to store.RepairStatus) bool {
	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// repairIsEditable returns true when the repair status allows structural changes
// (adding/removing steps and parts). Allowed: draft, in_progress.
func repairIsEditable(status store.RepairStatus) bool {
	return status == store.RepairStatusDraft || status == store.RepairStatusInProgress
}

// uuidToPgtype converts a google/uuid.UUID to a pgtype.UUID for store params
// that expect a nullable UUID (pgtype.UUID carries a Valid flag).
func uuidToPgtype(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

// Add to api/internal/handler/repair.go
// (Insert after CreateNote handler)

// ListNotes godoc
// @Summary      List notes on a repair
// @Description  Returns all timestamped notes for a repair, newest last.
// @Tags         repairs
// @Produce      json
// @Param        repairID  path      string  true  "Repair UUID"  format(uuid)
// @Success      200       {array}   SwaggerRepairNote
// @Failure      401       {object}  apierr.ErrorResponse
// @Failure      404       {object}  apierr.ErrorResponse
// @Security     BearerAuth
// @Router       /repairs/{repairID}/notes [get]
func (h *RepairHandler) ListNotes(c *gin.Context) {
	orgID, ok := orgIDFromCtx(c)
	if !ok {
		return
	}
	repairID, ok := parseUUIDParam(c, "repairID")
	if !ok {
		return
	}

	if _, err := h.q.GetRepair(c.Request.Context(), store.GetRepairParams{
		ID:    repairID,
		OrgID: orgID,
	}); err != nil {
		apierr.NotFound(c, "Repair not found.")
		return
	}

	notes, err := h.q.ListNotes(c.Request.Context(), repairID)
	if err != nil {
		h.log.Error("ListNotes", zap.Error(err))
		apierr.Internal(c)
		return
	}

	c.JSON(http.StatusOK, notes)
}

// ListHistory godoc
// @Summary      List repair history for a vehicle
// @Description  Returns completed repair snapshots from repair_history, ordered newest first.
// @Tags         vehicles
// @Produce      json
// @Param        vehicleID  path      string  true  "Vehicle UUID"  format(uuid)
// @Success      200        {array}   SwaggerRepairHistory
// @Failure      400        {object}  apierr.ErrorResponse
// @Failure      401        {object}  apierr.ErrorResponse
// @Failure      404        {object}  apierr.ErrorResponse
// @Security     BearerAuth
// @Router       /vehicles/{vehicleID}/history [get]
func (h *VehicleHandler) ListHistory(c *gin.Context) {
	orgID, ok := orgIDFromCtx(c)
	if !ok {
		return
	}
	vehicleID, ok := parseUUIDParam(c, "vehicleID")
	if !ok {
		return
	}

	if _, err := h.q.GetVehicle(c.Request.Context(), store.GetVehicleParams{
		ID:    vehicleID,
		OrgID: orgID,
	}); err != nil {
		apierr.NotFound(c, "Vehicle not found.")
		return
	}

	history, err := h.q.ListRepairHistory(c.Request.Context(), store.ListRepairHistoryParams{
		OrgID:     orgID,
		VehicleID: vehicleID,
	})
	if err != nil {
		h.log.Error("ListRepairHistory", zap.Error(err), zap.String("vehicle_id", vehicleID.String()))
		apierr.Internal(c)
		return
	}

	c.JSON(http.StatusOK, history)
}
