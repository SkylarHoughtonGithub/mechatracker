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

// garageSystemSummary is a per-system summary within a garage vehicle.
type garageSystemSummary struct {
	ID          string `json:"id"           example:"brakes"`
	Label       string `json:"label"        example:"Brakes"`
	RepairCount int64  `json:"repair_count" example:"3"`
	HasOpen     bool   `json:"has_open"     example:"true"`
}

// garageVehicle is the vehicle shape returned by the garage endpoint.
type garageVehicle struct {
	ID           string                `json:"id"                      example:"a44d8522-66b3-4146-af64-9db60c72470a"`
	Year         int16                 `json:"year"                    example:"2019"`
	Make         string                `json:"make"                    example:"Ford"`
	Model        string                `json:"model"                   example:"F-150"`
	Trim         *string               `json:"trim,omitempty"          example:"XLT"`
	VIN          *string               `json:"vin,omitempty"`
	Mileage      *int32                `json:"mileage,omitempty"       example:"84200"`
	UnitLabel    *string               `json:"unit_label,omitempty"`
	LastRepairAt *string               `json:"last_repair_at,omitempty" example:"2026-03-15"`
	OpenRepairs  int64                 `json:"open_repairs"            example:"1"`
	Systems      []garageSystemSummary `json:"systems"`
}

// garageStats is the org-level aggregate shown in the garage header.
type garageStats struct {
	TotalVehicles int64   `json:"total_vehicles" example:"4"`
	OpenRepairs   int64   `json:"open_repairs"   example:"2"`
	Repairs30d    int64   `json:"repairs_30d"    example:"5"`
	PartsSpend30d float64 `json:"parts_spend_30d" example:"342.50"`
}

// garageResponse is the full response body for GET /v1/garage.
type garageResponse struct {
	OrgName  string          `json:"org_name"`
	Vehicles []garageVehicle `json:"vehicles"`
	Stats    garageStats     `json:"stats"`
}

// GetGarage godoc
// @Summary      Aggregated garage view
// @Description  Returns all vehicles with per-system summaries and org-level stats in one call.
// @Tags         garage
// @Produce      json
// @Success      200  {object}  garageResponse
// @Failure      401  {object}  apierr.ErrorResponse
// @Failure      500  {object}  apierr.ErrorResponse
// @Security     BearerAuth
// @Router       /garage [get]
func (h *GarageHandler) GetGarage(c *gin.Context) {
	orgID, ok := orgIDFromCtx(c)
	if !ok {
		return
	}

	var (
		vehicleRows []store.GetGarageVehiclesRow
		statsRow    store.GetGarageStatsRow
		org         store.Organization
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

	g.Go(func() error {
		var err error
		org, err = h.q.GetOrgByID(ctx, orgID)
		return err
	})

	if err := g.Wait(); err != nil {
		h.log.Error("GetGarage", zap.Error(err), zap.String("org_id", orgID.String()))
		apierr.Internal(c)
		return
	}

	c.JSON(http.StatusOK, garageResponse{
		OrgName:  org.Name,
		Vehicles: collapseVehicleRows(vehicleRows),
		Stats:    mapGarageStats(statsRow),
	})
}

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
			if row.LastRepairAt != nil {
				if ts, ok := row.LastRepairAt.(pgtype.Timestamptz); ok && ts.Valid {
					t := ts.Time.Format("2006-01-02")
					v.LastRepairAt = &t
				}
			}
			index[vid] = len(vehicles)
			vehicles = append(vehicles, v)
		}

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
