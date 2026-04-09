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

// ── GET /v1/repairs ───────────────────────────────────────────────────────────
// Returns active (in_progress) repairs assigned to or created by the caller.

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

// ── POST /v1/repairs ──────────────────────────────────────────────────────────

type createRepairReq struct {
	VehicleID  string  `json:"vehicle_id"  binding:"required,uuid"`
	Title      string  `json:"title"       binding:"required,max=200"`
	System     string  `json:"system"      binding:"required,max=100"`
	AssignedTo *string `json:"assigned_to" binding:"omitempty,uuid"`
	TemplateID *string `json:"template_id" binding:"omitempty,uuid"`
}

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

// ── GET /v1/repairs/:repairID ─────────────────────────────────────────────────
// Returns the repair record plus its steps and parts in one response.
// Steps and parts are fetched in parallel.

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

// ── PATCH /v1/repairs/:repairID ───────────────────────────────────────────────
// Handles two distinct update operations:
//   - status change (status field present) → UpdateRepairStatus
//   - field edits (title, system, assigned_to) → UpdateRepair
//
// When transitioning to completed, an immutable history snapshot is written.
// The snapshot write is best-effort — a failure logs but does not roll back
// the status update, since the status transition already committed.

type updateRepairReq struct {
	// Status transition — mutually exclusive with field edits.
	Status *string `json:"status" binding:"omitempty,oneof=draft in_progress pending_parts completed"`

	// Field edits — all optional.
	Title      *string `json:"title"       binding:"omitempty,max=200"`
	System     *string `json:"system"      binding:"omitempty,max=100"`
	AssignedTo *string `json:"assigned_to" binding:"omitempty,uuid"`
}

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
		// Fetch current state to validate the transition.
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

	// Field edit path.
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

// ── writeRepairHistory ────────────────────────────────────────────────────────
// Called when a repair transitions to completed. Fetches the vehicle, assignee,
// steps, parts, notes, and resources in parallel then writes a single immutable
// snapshot row to repair_history.

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

	// Fetch assignee only if assigned_to is set.
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

	// Build human-readable snapshot strings.
	vehicleSnapshot := buildVehicleSnapshot(vehicle)
	var assignedSnapshot *string
	if assignee != nil {
		s := assignee.Name
		assignedSnapshot = &s
	}

	// Marshal list snapshots to JSON.
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

// buildVehicleSnapshot produces the human-readable vehicle string frozen into
// repair_history at completion time. Example: "2018 Ford F-150 XLT — Unit #07"
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

// ── POST /v1/repairs/:repairID/steps ─────────────────────────────────────────
// Adds a step to a repair. Only allowed when status is draft or in_progress.

type createStepReq struct {
	Position   int16   `json:"position"    binding:"required,min=1"`
	Title      string  `json:"title"       binding:"required,max=200"`
	Notes      *string `json:"notes"`
	TorqueSpec *string `json:"torque_spec"`
}

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

// ── DELETE /v1/repairs/:repairID/steps/:stepID ────────────────────────────────
// Removes a step. Only allowed when status is draft or in_progress.

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

// ── POST /v1/repairs/:repairID/steps/:stepID/complete ─────────────────────────
// Toggles step completion. If already complete → uncomplete; if incomplete → complete.
// After toggling, recalculates progress_pct and total_parts_cost on the repair.

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

// ── POST /v1/repairs/:repairID/notes ─────────────────────────────────────────

type createNoteReq struct {
	Body string `json:"body" binding:"required,min=1,max=5000"`
}

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

// ── POST /v1/repairs/:repairID/photos ─────────────────────────────────────────
// Returns a presigned upload URL. The client uploads directly to Spaces —
// the API never proxies the bytes.
//
// TODO (Phase 2): wire up DigitalOcean Spaces presigned URLs once
// SPACES_ENDPOINT / SPACES_BUCKET / SPACES_ACCESS_KEY / SPACES_SECRET_KEY
// are configured.

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
		"message": "Photo upload not yet implemented — wire up DigitalOcean Spaces in Phase 2.",
	})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// validTransitions defines the allowed status progressions for a repair.
// Any transition not listed here is rejected with a 400.
var validTransitions = map[store.RepairStatus][]store.RepairStatus{
	store.RepairStatusDraft:        {store.RepairStatusInProgress},
	store.RepairStatusInProgress:   {store.RepairStatusPendingParts, store.RepairStatusCompleted},
	store.RepairStatusPendingParts: {store.RepairStatusInProgress, store.RepairStatusCompleted},
	store.RepairStatusCompleted:    {}, // terminal — no transitions out
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
