package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/mechatracker/api/internal/apierr"
	"github.com/mechatracker/api/internal/store"
)

// GarageHandler handles the aggregated garage view.
type GarageHandler struct {
	q   store.Querier
	log *zap.Logger
}

func NewGarageHandler(q store.Querier, log *zap.Logger) *GarageHandler {
	return &GarageHandler{q: q, log: log}
}

// ── Response types ────────────────────────────────────────────────────────────

type garageSystemSummary struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	RepairCount int64  `json:"repair_count"`
	HasOpen     bool   `json:"has_open"`
}

type garageVehicle struct {
	ID           string                `json:"id"`
	Year         int16                 `json:"year"`
	Make         string                `json:"make"`
	Model        string                `json:"model"`
	Trim         *string               `json:"trim,omitempty"`
	VIN          *string               `json:"vin,omitempty"`
	Mileage      *int32                `json:"mileage,omitempty"`
	UnitLabel    *string               `json:"unit_label,omitempty"`
	LastRepairAt *string               `json:"last_repair_at,omitempty"`
	OpenRepairs  int64                 `json:"open_repairs"`
	Systems      []garageSystemSummary `json:"systems"`
}

type garageStats struct {
	TotalVehicles int64   `json:"total_vehicles"`
	OpenRepairs   int64   `json:"open_repairs"`
	Repairs30d    int64   `json:"repairs_30d"`
	PartsSpend30d float64 `json:"parts_spend_30d"`
}

type garageResponse struct {
	Vehicles []garageVehicle `json:"vehicles"`
	Stats    garageStats     `json:"stats"`
}

// ── Handler ───────────────────────────────────────────────────────────────────

// GetGarage godoc
// @Summary      Aggregated garage view
// @Description  Returns all vehicles with per-system summaries and org-level stats in one call.
// @Tags         garage
// @Produce      json
// @Success      200  {object}  garageResponse
// @Failure      500  {object}  apierr.APIError
// @Router       /v1/garage [get]
func (h *GarageHandler) GetGarage(c *gin.Context) {
	orgID, ok := orgIDFromCtx(c)
	if !ok {
		return
	}

	var (
		vehicleRows []store.GetGarageVehiclesRow
		statsRow    store.GetGarageStatsRow
	)

	g, ctx := errgroup.WithContext(c.Request.Context())

	g.Go(func() error {
		var err error
		vehicleRows, err = h.q.GetGarageVehicles(ctx, orgID)
		return err
	})

	g.Go(func() error {
		var err error
		statsRow, err = h.q.GetGarageStats(ctx, orgID)
		return err
	})

	if err := g.Wait(); err != nil {
		h.log.Error("GetGarage", zap.Error(err), zap.String("org_id", orgID.String()))
		apierr.Internal(c)
		return
	}

	c.JSON(http.StatusOK, garageResponse{
		Vehicles: collapseVehicleRows(vehicleRows),
		Stats:    mapGarageStats(statsRow),
	})
}

// ── Collapse helpers ──────────────────────────────────────────────────────────

// collapseVehicleRows turns the flat JOIN rows (one per vehicle+system) into
// the nested vehicle → []systems shape the frontend expects.
func collapseVehicleRows(rows []store.GetGarageVehiclesRow) []garageVehicle {
	index := make(map[string]int)
	vehicles := []garageVehicle{}

	for _, row := range rows {
		vid := row.VehicleID.String()

		if _, exists := index[vid]; !exists {
			v := garageVehicle{
				ID:          vid,
				Year:        row.Year,
				Make:        row.Make,
				Model:       row.Model,
				Trim:        row.Trim,
				VIN:         row.Vin,
				Mileage:     row.Mileage,
				UnitLabel:   row.UnitLabel,
				OpenRepairs: row.OpenRepairCount,
				Systems:     []garageSystemSummary{},
			}
			// LastRepairAt is interface{} (scalar subquery) — pgx returns
			// pgtype.Timestamptz at runtime. Type-assert to extract the date.
			if row.LastRepairAt != nil {
				if ts, ok := row.LastRepairAt.(pgtype.Timestamptz); ok && ts.Valid {
					t := ts.Time.Format("2006-01-02")
					v.LastRepairAt = &t
				}
			}
			index[vid] = len(vehicles)
			vehicles = append(vehicles, v)
		}

		// SystemKey is *string — NULL for vehicles with no repairs.
		if row.SystemKey != nil && *row.SystemKey != "" {
			var repairCount int64
			if v, ok := row.SystemRepairCount.(int64); ok {
				repairCount = v
			}
			var hasOpen bool
			if v, ok := row.SystemHasOpen.(bool); ok {
				hasOpen = v
			}
			i := index[vid]
			vehicles[i].Systems = append(vehicles[i].Systems, garageSystemSummary{
				ID:          *row.SystemKey,
				Label:       systemLabel(*row.SystemKey),
				RepairCount: repairCount,
				HasOpen:     hasOpen,
			})
		}
	}

	return vehicles
}

func mapGarageStats(row store.GetGarageStatsRow) garageStats {
	return garageStats{
		TotalVehicles: row.TotalVehicles,
		OpenRepairs:   row.OpenRepairs,
		Repairs30d:    row.Repairs30d,
		PartsSpend30d: row.PartsSpend30d.InexactFloat64(),
	}
}

// systemLabel converts a system key like "brakes" to a display label "Brakes".
func systemLabel(key string) string {
	labels := map[string]string{
		"brakes":       "Brakes",
		"engine":       "Engine",
		"transmission": "Transmission",
		"suspension":   "Suspension",
		"electrical":   "Electrical",
		"hvac":         "HVAC",
		"exhaust":      "Exhaust",
		"fuel":         "Fuel System",
		"cooling":      "Cooling",
		"steering":     "Steering",
		"tires":        "Tires & Wheels",
		"body":         "Body & Interior",
	}
	if l, ok := labels[key]; ok {
		return l
	}
	return key
}
