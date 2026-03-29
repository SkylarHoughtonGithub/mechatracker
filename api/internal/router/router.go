// Package router wires up the Gin engine: global middleware, versioned route
// groups, and handler registration. Add new route groups here as features
// are built out. Handlers themselves live in internal/handlers/*.
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/mechatracker/api/internal/apierr"
	"github.com/mechatracker/api/internal/config"
	"github.com/mechatracker/api/internal/middleware"
)

// New builds and returns the configured Gin engine.
func New(cfg *config.Config, log *zap.Logger) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New() // don't use gin.Default() — we supply our own middleware

	// ── Global middleware (every request) ────────────────────────────────────
	r.Use(
		middleware.RequestID,
		middleware.Recovery(log),
		middleware.Logger(log),
	)

	// ── Health / readiness (no auth) ─────────────────────────────────────────
	r.GET("/health", handleHealth)
	r.GET("/ready", handleReady)

	// ── API v1 ────────────────────────────────────────────────────────────────
	v1 := r.Group("/v1")
	v1.Use(middleware.Auth(cfg, log))
	{
		registerVehicleRoutes(v1)
		registerRepairRoutes(v1)
		registerGarageRoutes(v1)

		// Org-scoped routes — require an active Clerk org context.
		org := v1.Group("/org")
		org.Use(middleware.RequireOrg)
		{
			registerOrgRoutes(org)
		}
	}

	// ── 404 / 405 handlers ────────────────────────────────────────────────────
	r.NoRoute(func(c *gin.Context) {
		apierr.NotFound(c, "The requested route does not exist.")
	})
	r.NoMethod(func(c *gin.Context) {
		apierr.BadRequest(c, "Method not allowed on this route.")
	})

	return r
}

// ── Health handlers ───────────────────────────────────────────────────────────

func handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleReady is a richer check — add DB ping here once the store layer exists.
func handleReady(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// ── Route group stubs — replace with real handlers as you build each feature ──

func registerVehicleRoutes(rg *gin.RouterGroup) {
	v := rg.Group("/vehicles")
	v.GET("", stub("list vehicles"))
	v.POST("", stub("create vehicle"))
	v.GET("/:vehicleID", stub("get vehicle"))
	v.PATCH("/:vehicleID", stub("update vehicle"))
	v.DELETE("/:vehicleID", stub("delete vehicle"))
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

// stub returns a placeholder handler that responds 501 until real handlers are
// wired in. Replace with actual handler functions as each feature is built.
func stub(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"message": name + " — not yet implemented",
		})
	}
}
