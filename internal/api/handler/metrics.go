package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsHandler Prometheus metrics 处理器
type MetricsHandler struct{}

// NewMetricsHandler 创建 metrics 处理器
func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{}
}

// Handler 返回 Prometheus HTTP 处理器
func (h *MetricsHandler) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	}
}
