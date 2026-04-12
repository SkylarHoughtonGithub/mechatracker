package router

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	"github.com/mechatracker/api/internal/apierr"
	"github.com/mechatracker/api/internal/config"
	"github.com/mechatracker/api/internal/handler"
	"github.com/mechatracker/api/internal/middleware"
	"github.com/mechatracker/api/internal/store"
)

// New builds and returns the configured Gin engine.
func New(cfg *config.Config, log *zap.Logger, q store.Querier) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:3000",
			"http://localhost:8080",
			"http://web.localhost:8080",
		},
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.Use(
		middleware.RequestID,
		middleware.Recovery(log),
		middleware.Logger(log),
	)

	r.GET("/health", handleHealth)
	r.GET("/ready", handleReady)

	// ── Swagger UI (non-production only) ─────────────────────
	if !cfg.IsProduction() {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// ── Handlers ─────────────────────────────────────────────
	vh := handler.NewVehicleHandler(q, log)
	gh := handler.NewGarageHandler(q, log)
	rh := handler.NewRepairHandler(q, log)
	ph := handler.NewPartsHandler(q, log)

	v1 := r.Group("/v1")
	v1.Use(middleware.Auth(cfg, log))
	v1.Use(middleware.UserLookup(q, log))
	{
		registerVehicleRoutes(v1, vh)
		registerRepairRoutes(v1, rh, ph)
		registerGarageRoutes(v1, gh)

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

func handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleReady(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

func registerVehicleRoutes(rg *gin.RouterGroup, h *handler.VehicleHandler) {
	v := rg.Group("/vehicles")
	v.GET("", h.List)
	v.POST("", h.Create)
	v.GET("/:vehicleID", h.Get)
	v.PATCH("/:vehicleID", h.Update)
	v.DELETE("/:vehicleID", h.Delete)
	v.GET("/:vehicleID/repairs", h.ListRepairs)
}

func registerRepairRoutes(rg *gin.RouterGroup, h *handler.RepairHandler, ph *handler.PartsHandler) {
	r := rg.Group("/repairs")
	r.GET("", h.List)
	r.POST("", h.Create)
	r.GET("/:repairID", h.Get)
	r.PATCH("/:repairID", h.Update)
	r.POST("/:repairID/steps", h.CreateStep)
	r.POST("/:repairID/steps/:stepID/complete", h.CompleteStep)
	r.DELETE("/:repairID/steps/:stepID", h.DeleteStep)
	r.POST("/:repairID/parts", ph.Create)
	r.PATCH("/:repairID/parts/:partID", ph.Update)
	r.PATCH("/:repairID/parts/:partID/status", ph.UpdateStatus)
	r.DELETE("/:repairID/parts/:partID", ph.Delete)
	r.POST("/:repairID/notes", h.CreateNote)
	r.POST("/:repairID/photos", h.UploadPhoto)
}

func registerGarageRoutes(rg *gin.RouterGroup, h *handler.GarageHandler) {
	g := rg.Group("/garage")
	g.GET("", h.GetGarage)
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
