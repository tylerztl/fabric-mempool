package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/tylerztl/fabric-mempool/conf"
	"net/http"
)

type RestHandler struct {
	distributeConfig *conf.DistributeConfig
	handler          *Handler
}

func NewRestHandler(config *conf.DistributeConfig, handler *Handler) *RestHandler {
	return &RestHandler{
		distributeConfig: config,
		handler:          handler,
	}
}

// changeDistribute change distribution type
func (h *RestHandler) changeDistribute(ctx *gin.Context) {
	config := conf.DistributeConfig{}
	if err := ctx.ShouldBindJSON(config); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "params not valid"})
		return
	}
	h.handler.ChangeDistribute(&config)
	ctx.JSON(http.StatusOK, gin.H{})
}

// changeSortSwitch open/close mempool tx sort by fee.
func (h *RestHandler) changeSortSwitch(ctx *gin.Context) {
	config := conf.SortConfig{}
	if err := ctx.ShouldBindJSON(config); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "params not valid"})
		return
	}
	h.handler.ChangeSortSwitch(&config)
	ctx.JSON(http.StatusOK, gin.H{})
}

// changeSortSwitch open/close mempool tx sort by fee.
func (h *RestHandler) changeOrdererCapacity(ctx *gin.Context) {
	config := conf.OrdererCapacityConfig{}
	if err := ctx.ShouldBindJSON(config); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "params not valid"})
		return
	}
	if err := h.handler.ChangeOrdererCapacity(&config); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{})
}

// getOrdererLog get one orderer log with special name
func (h *RestHandler) getOrdererLog(ctx *gin.Context) {
	sender := ctx.Param("sender")
	log, err := h.handler.GetOrdererLog(sender)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"msg": "operator success", "data": log})
}

// Register register route info to gin
func (h *RestHandler) Register(r *gin.Engine) {
	r.POST("/distribute", h.changeDistribute)
	r.POST("/sort", h.changeSortSwitch)
	r.POST("/capacity", h.changeOrdererCapacity)
	r.GET("/orderer/:sender", h.getOrdererLog)
}
