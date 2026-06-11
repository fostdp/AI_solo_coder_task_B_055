package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"stone-relic-monitor/internal/algorithms"
	"stone-relic-monitor/internal/config"
	"stone-relic-monitor/internal/db"
	"stone-relic-monitor/internal/models"
	"stone-relic-monitor/internal/services"
)

type AdvancedCleaningHandler struct {
	cfg       *config.Config
	db        *db.ClickHouse
	robotSvc  *services.RobotSimulationService
}

func NewAdvancedCleaningHandler(cfg *config.Config, db *db.ClickHouse) *AdvancedCleaningHandler {
	return &AdvancedCleaningHandler{
		cfg:      cfg,
		db:       db,
		robotSvc: services.NewRobotSimulationService(),
	}
}

func (h *AdvancedCleaningHandler) PlanTSPPath(c *gin.Context) {
	var req models.TSPPathRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Points) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "points cannot be empty"})
		return
	}

	result := algorithms.SolveTSP(&req)
	c.JSON(http.StatusOK, result)
}

func (h *AdvancedCleaningHandler) PredictRoughness(c *gin.Context) {
	var req models.RoughnessPredictionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.MineralComposition == nil {
		req.MineralComposition = map[string]float32{
			"calcium_sulfate": 0.6,
			"calcite":         0.25,
			"dolomite":        0.1,
			"silicate":        0.05,
		}
	}

	result := algorithms.PredictRoughness(&req)
	c.JSON(http.StatusOK, result)
}

func (h *AdvancedCleaningHandler) PredictRescaling(c *gin.Context) {
	var req models.RescalingPredictionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.HistoryData) == 0 {
		req.HistoryData = []float32{0.02, 0.03, 0.04, 0.05, 0.06, 0.07, 0.08, 0.09, 0.10}
	}

	result := algorithms.PredictRescaling(&req)
	c.JSON(http.StatusOK, result)
}

func (h *AdvancedCleaningHandler) SimulateRobot(c *gin.Context) {
	var req models.RobotSimulationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Path) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path cannot be empty"})
		return
	}

	result := h.robotSvc.Simulate(&req)
	c.JSON(http.StatusOK, result)
}

func (h *AdvancedCleaningHandler) RegisterRoutes(r *gin.RouterGroup) {
	adv := r.Group("/advanced")
	{
		adv.POST("/plan-tsp-path", h.PlanTSPPath)
		adv.POST("/predict-roughness", h.PredictRoughness)
		adv.POST("/predict-rescaling", h.PredictRescaling)
		adv.POST("/simulate-robot", h.SimulateRobot)
	}
}
