package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"github.com/mechatracker/api/internal/apierr"
	"github.com/mechatracker/api/internal/store"
)

// PartsHandler handles all part-related endpoints.
// Parts are always scoped to a repair — there is no top-level parts resource.
type PartsHandler struct {
	q   store.Querier
	log *zap.Logger
}

func NewPartsHandler(q store.Querier, log *zap.Logger) *PartsHandler {
	return &PartsHandler{q: q, log: log}
}

// ── POST /v1/repairs/:repairID/parts ─────────────────────────────────────────
// Adds a part to a repair. Only allowed when status is draft or in_progress.
// Cost is accepted as a decimal string e.g. "29.99" to avoid float precision issues.

type createPartReq struct {
	Name       string  `json:"name"        binding:"required,max=200"`
	PartNumber *string `json:"part_number"`
	Supplier   *string `json:"supplier"`
	Cost       *string `json:"cost"`
	Notes      *string `json:"notes"`
}

func (h *PartsHandler) Create(c *gin.Context) {
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
		apierr.BadRequest(c, "Parts can only be added to repairs with status draft or in_progress.")
		return
	}

	var req createPartReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(c, err.Error())
		return
	}

	cost, ok := parseCost(c, req.Cost)
	if !ok {
		return
	}

	part, err := h.q.CreatePart(c.Request.Context(), store.CreatePartParams{
		RepairID:   repairID,
		Name:       req.Name,
		PartNumber: req.PartNumber,
		Supplier:   req.Supplier,
		Status:     store.PartStatusNeeded, // always starts as needed
		Cost:       cost,
		Notes:      req.Notes,
	})
	if err != nil {
		h.log.Error("CreatePart", zap.Error(err))
		apierr.Internal(c)
		return
	}

	c.JSON(http.StatusCreated, part)
}

// ── PATCH /v1/repairs/:repairID/parts/:partID ────────────────────────────────
// Updates part fields (name, part_number, supplier, cost, notes).
// Allowed on any non-completed repair so techs can update cost and supplier
// while parts are being sourced.

type updatePartReq struct {
	Name       *string `json:"name"        binding:"omitempty,max=200"`
	PartNumber *string `json:"part_number"`
	Supplier   *string `json:"supplier"`
	Cost       *string `json:"cost"`
	Notes      *string `json:"notes"`
}

func (h *PartsHandler) Update(c *gin.Context) {
	orgID, ok := orgIDFromCtx(c)
	if !ok {
		return
	}
	repairID, ok := parseUUIDParam(c, "repairID")
	if !ok {
		return
	}
	partID, ok := parseUUIDParam(c, "partID")
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

	if repair.Status == store.RepairStatusCompleted {
		apierr.BadRequest(c, "Parts cannot be edited on a completed repair.")
		return
	}

	var req updatePartReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(c, err.Error())
		return
	}

	cost, ok := parseCost(c, req.Cost)
	if !ok {
		return
	}

	part, err := h.q.UpdatePart(c.Request.Context(), store.UpdatePartParams{
		ID:         partID,
		RepairID:   repairID,
		Name:       req.Name,
		PartNumber: req.PartNumber,
		Supplier:   req.Supplier,
		Cost:       cost,
		Notes:      req.Notes,
	})
	if err != nil {
		apierr.NotFound(c, "Part not found.")
		return
	}

	c.JSON(http.StatusOK, part)
}

// ── PATCH /v1/repairs/:repairID/parts/:partID/status ─────────────────────────
// Transitions a part through its lifecycle: needed → ordered → in_stock → installed.
// Allowed on any non-completed repair.

// validPartTransitions defines the allowed part status progressions.
var validPartTransitions = map[store.PartStatus][]store.PartStatus{
	store.PartStatusNeeded:    {store.PartStatusOrdered, store.PartStatusInStock},
	store.PartStatusOrdered:   {store.PartStatusInStock},
	store.PartStatusInStock:   {store.PartStatusInstalled},
	store.PartStatusInstalled: {}, // terminal
}

func isValidPartTransition(from, to store.PartStatus) bool {
	allowed, ok := validPartTransitions[from]
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

type updatePartStatusReq struct {
	Status string `json:"status" binding:"required,oneof=needed ordered in_stock installed"`
}

func (h *PartsHandler) UpdateStatus(c *gin.Context) {
	orgID, ok := orgIDFromCtx(c)
	if !ok {
		return
	}
	repairID, ok := parseUUIDParam(c, "repairID")
	if !ok {
		return
	}
	partID, ok := parseUUIDParam(c, "partID")
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

	if repair.Status == store.RepairStatusCompleted {
		apierr.BadRequest(c, "Part status cannot be changed on a completed repair.")
		return
	}

	var req updatePartStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(c, err.Error())
		return
	}

	// Fetch current parts to find the one we're updating and check current status.
	parts, err := h.q.ListParts(c.Request.Context(), repairID)
	if err != nil {
		h.log.Error("ListParts for status check", zap.Error(err))
		apierr.Internal(c)
		return
	}

	var currentStatus store.PartStatus
	found := false
	for _, p := range parts {
		if p.ID == partID {
			currentStatus = p.Status
			found = true
			break
		}
	}
	if !found {
		apierr.NotFound(c, "Part not found.")
		return
	}

	to := store.PartStatus(req.Status)
	if !isValidPartTransition(currentStatus, to) {
		apierr.BadRequest(c, "Cannot transition part from \""+string(currentStatus)+"\" to \""+string(to)+"\".")
		return
	}

	part, err := h.q.UpdatePartStatus(c.Request.Context(), store.UpdatePartStatusParams{
		ID:       partID,
		RepairID: repairID,
		Status:   to,
	})
	if err != nil {
		apierr.NotFound(c, "Part not found.")
		return
	}

	c.JSON(http.StatusOK, part)
}

// ── DELETE /v1/repairs/:repairID/parts/:partID ───────────────────────────────
// Removes a part. Only allowed when status is draft or in_progress.

func (h *PartsHandler) Delete(c *gin.Context) {
	orgID, ok := orgIDFromCtx(c)
	if !ok {
		return
	}
	repairID, ok := parseUUIDParam(c, "repairID")
	if !ok {
		return
	}
	partID, ok := parseUUIDParam(c, "partID")
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
		apierr.BadRequest(c, "Parts can only be removed from repairs with status draft or in_progress.")
		return
	}

	if err := h.q.DeletePart(c.Request.Context(), store.DeletePartParams{
		ID:       partID,
		RepairID: repairID,
	}); err != nil {
		h.log.Error("DeletePart", zap.Error(err))
		apierr.Internal(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": true, "id": partID})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// parseCost converts an optional decimal string like "29.99" into a pgtype.Numeric.
// Returns a null Numeric when cost is nil. Aborts with 400 on malformed input.
func parseCost(c *gin.Context, cost *string) (pgtype.Numeric, bool) {
	var n pgtype.Numeric
	if cost == nil {
		return n, true // null — no cost provided
	}
	if err := n.Scan(*cost); err != nil {
		apierr.BadRequest(c, "Invalid cost: must be a decimal number e.g. \"29.99\".")
		return n, false
	}
	return n, true
}
