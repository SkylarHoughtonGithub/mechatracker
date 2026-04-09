package handler

import (
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
// Both are combined into one endpoint to keep the client surface simple.

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
		// Status transition path.
		repair, err := h.q.UpdateRepairStatus(c.Request.Context(), store.UpdateRepairStatusParams{
			ID:     repairID,
			OrgID:  orgID,
			Status: store.RepairStatus(*req.Status),
		})
		if err != nil {
			apierr.NotFound(c, "Repair not found.")
			return
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

// ── POST /v1/repairs/:repairID/steps/:stepID/complete ─────────────────────────
// Toggles step completion. If already complete → uncomplete; if incomplete → complete.
// After toggling, recalculates progress_pct and total_parts_cost on the repair.
//
// Route note: the router currently registers this as complete-only. Wire up a
// DELETE or separate POST /uncomplete if you want explicit uncomplete from the client.
// For now a single endpoint that reads the current state and toggles is simpler.

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

	// Verify the repair belongs to this org before touching steps.
	repair, err := h.q.GetRepair(c.Request.Context(), store.GetRepairParams{
		ID:    repairID,
		OrgID: orgID,
	})
	if err != nil {
		apierr.NotFound(c, "Repair not found.")
		return
	}

	// Fetch the step to check its current state.
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

	// Recalculate progress_pct.
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

	// Refresh total_parts_cost while we're here.
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

	// Suppress unused variable warning — repair was fetched for auth, not used in response.
	_ = repair

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

	// Verify repair ownership.
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
// are configured. For now this returns a 501 with a clear message.

func (h *RepairHandler) UploadPhoto(c *gin.Context) {
	orgID, ok := orgIDFromCtx(c)
	if !ok {
		return
	}
	repairID, ok := parseUUIDParam(c, "repairID")
	if !ok {
		return
	}

	// Verify repair ownership before issuing any URL.
	if _, err := h.q.GetRepair(c.Request.Context(), store.GetRepairParams{
		ID:    repairID,
		OrgID: orgID,
	}); err != nil {
		apierr.NotFound(c, "Repair not found.")
		return
	}

	// TODO: generate presigned PUT URL via aws-sdk-go-v2 + DO Spaces endpoint.
	// Shape of the eventual response:
	//   { "upload_url": "https://...", "storage_key": "repairs/<id>/photos/<uuid>.jpg", "expires_in": 900 }
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "Photo upload not yet implemented — wire up DigitalOcean Spaces in Phase 2.",
	})
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

// ── Helpers ───────────────────────────────────────────────────────────────────

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
