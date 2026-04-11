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

// createPartReq is the request body for adding a part to a repair.
type createPartReq struct {
	Name       string  `json:"name"        binding:"required,max=200" example:"Brake pad set"`
	PartNumber *string `json:"part_number"                            example:"D1060-8J090"`
	Supplier   *string `json:"supplier"                               example:"RockAuto"`
	Cost       *string `json:"cost"                                   example:"34.99"`
	Notes      *string `json:"notes"`
}

// Create godoc
// @Summary      Add a part to a repair
// @Description  Creates a part entry on a repair, starting in `needed` status. Cost is a decimal string e.g. "34.99".
// @Tags         parts
// @Accept       json
// @Produce      json
// @Param        repairID  path      string          true  "Repair UUID"  format(uuid)
// @Param        body      body      createPartReq   true  "Part details"
// @Success      201       {object}  SwaggerPart
// @Failure      400       {object}  apierr.ErrorResponse
// @Failure      401       {object}  apierr.ErrorResponse
// @Failure      404       {object}  apierr.ErrorResponse
// @Failure      500       {object}  apierr.ErrorResponse
// @Security     BearerAuth
// @Router       /repairs/{repairID}/parts [post]
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
		Status:     store.PartStatusNeeded,
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

// updatePartReq is the request body for updating a part's fields.
type updatePartReq struct {
	Name       *string `json:"name"        binding:"omitempty,max=200" example:"Brake pad set — premium"`
	PartNumber *string `json:"part_number"`
	Supplier   *string `json:"supplier"                                example:"NAPA"`
	Cost       *string `json:"cost"                                    example:"49.99"`
	Notes      *string `json:"notes"`
}

// Update godoc
// @Summary      Update a part
// @Description  Updates part fields (name, part_number, supplier, cost, notes). Allowed on any non-completed repair.
// @Tags         parts
// @Accept       json
// @Produce      json
// @Param        repairID  path      string          true  "Repair UUID"  format(uuid)
// @Param        partID    path      string          true  "Part UUID"    format(uuid)
// @Param        body      body      updatePartReq   true  "Fields to update"
// @Success      200       {object}  SwaggerPart
// @Failure      400       {object}  apierr.ErrorResponse
// @Failure      401       {object}  apierr.ErrorResponse
// @Failure      404       {object}  apierr.ErrorResponse
// @Security     BearerAuth
// @Router       /repairs/{repairID}/parts/{partID} [patch]
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

// updatePartStatusReq is the request body for advancing a part through its lifecycle.
type updatePartStatusReq struct {
	// Status must follow the allowed progression: needed → ordered → in_stock → installed.
	Status string `json:"status" binding:"required,oneof=needed ordered in_stock installed" example:"ordered"`
}

var validPartTransitions = map[store.PartStatus][]store.PartStatus{
	store.PartStatusNeeded:    {store.PartStatusOrdered, store.PartStatusInStock},
	store.PartStatusOrdered:   {store.PartStatusInStock},
	store.PartStatusInStock:   {store.PartStatusInstalled},
	store.PartStatusInstalled: {},
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

// UpdateStatus godoc
// @Summary      Advance a part's status
// @Description  Transitions a part through its lifecycle: needed → ordered → in_stock → installed. Invalid transitions are rejected.
// @Tags         parts
// @Accept       json
// @Produce      json
// @Param        repairID  path      string                true  "Repair UUID"  format(uuid)
// @Param        partID    path      string                true  "Part UUID"    format(uuid)
// @Param        body      body      updatePartStatusReq   true  "New status"
// @Success      200       {object}  SwaggerPart
// @Failure      400       {object}  apierr.ErrorResponse
// @Failure      401       {object}  apierr.ErrorResponse
// @Failure      404       {object}  apierr.ErrorResponse
// @Security     BearerAuth
// @Router       /repairs/{repairID}/parts/{partID}/status [patch]
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

// Delete godoc
// @Summary      Delete a part
// @Description  Removes a part from a repair. Only allowed when status is draft or in_progress.
// @Tags         parts
// @Produce      json
// @Param        repairID  path      string  true  "Repair UUID"  format(uuid)
// @Param        partID    path      string  true  "Part UUID"    format(uuid)
// @Success      200       {object}  SwaggerDeletedResponse  "deleted: true, id: <uuid>"
// @Failure      400       {object}  apierr.ErrorResponse
// @Failure      401       {object}  apierr.ErrorResponse
// @Failure      404       {object}  apierr.ErrorResponse
// @Security     BearerAuth
// @Router       /repairs/{repairID}/parts/{partID} [delete]
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
		return n, true
	}
	if err := n.Scan(*cost); err != nil {
		apierr.BadRequest(c, "Invalid cost: must be a decimal number e.g. \"29.99\".")
		return n, false
	}
	return n, true
}
