package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/mechatracker/api/internal/apierr"
	"github.com/mechatracker/api/internal/config"
	"github.com/mechatracker/api/internal/handler"
	"github.com/mechatracker/api/internal/middleware"
	"github.com/mechatracker/api/internal/store"
)

// New builds and returns the configured Gin engine.
// q is the sqlc Querier — passed in so main.go owns the DB pool lifecycle.
func New(cfg *config.Config, log *zap.Logger, q store.Querier) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	r.Use(
		middleware.RequestID,
		middleware.Recovery(log),
		middleware.Logger(log),
	)

	r.GET("/health", handleHealth)
	r.GET("/ready", handleReady)

	// ── Handlers ─────────────────────────────────────────────
	vh := handler.NewVehicleHandler(q, log)

	v1 := r.Group("/v1")
	v1.Use(middleware.Auth(cfg, log))
	{
		registerVehicleRoutes(v1, vh)
		registerRepairRoutes(v1)
		registerGarageRoutes(v1)

		org := v1.Group("/org")
		org.Use(middleware.RequireOrg)
		{
			registerOrgRoutes(org)
		}
	}

	r.NoRoute(func(c *gin.Context) {
		apierr.NotFound(c, "The requested route does not exist.")
	})
	r.NoMethod(func(c *gin.Context) {
		apierr.BadRequest(c, "Method not allowed on this route.")
	})

	return r
}

// ── Health ────────────────────────────────────────────────────

func handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleReady(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// ── Route registration ────────────────────────────────────────

// registerVehicleRoutes wires the real VehicleHandler.
// Stub replaced.
func registerVehicleRoutes(rg *gin.RouterGroup, h *handler.VehicleHandler) {
	v := rg.Group("/vehicles")
	v.GET("", h.List)
	v.POST("", h.Create)
	v.GET("/:vehicleID", h.Get)
	v.PATCH("/:vehicleID", h.Update)
	v.DELETE("/:vehicleID", h.Delete)
}

func registerRepairRoutes(rg *gin.RouterGroup) {
	r := rg.Group("/repairs")
	r.GET("", stub("list repairs"))
	r.POST("", stub("create repair"))
	r.GET("/:repairID", stub("get repair"))
	r.PATCH("/:repairID", stub("update repair"))
	r.POST("/:repairID/steps/:stepID/complete", stub("complete step"))
	r.POST("/:repairID/notes", stub("add note"))
	r.POST("/:repairID/photos", stub("upload photo"))
}

func registerGarageRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/garage")
	g.GET("", stub("get garage"))
}

func registerOrgRoutes(rg *gin.RouterGroup) {
	rg.GET("/members", stub("list org members"))
	rg.PATCH("/members/:userID/role", stub("update member role"))
}

func stub(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"message": name + " — not yet implemented",
		})
	}
}
