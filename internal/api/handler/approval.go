package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ApprovalHandler 审批处理器
type ApprovalHandler struct {
	db *gorm.DB
}

// NewApprovalHandler 创建处理器
func NewApprovalHandler(db *gorm.DB) *ApprovalHandler {
	return &ApprovalHandler{db: db}
}

// Approval 审批信息
type Approval struct {
	ID         uint   `json:"id"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	Applicant  string `json:"applicant"`
	Approver   string `json:"approver"`
	ApplyTime  string `json:"apply_time"`
	ApproveTime string `json:"approve_time"`
}

// PendingCount 待审批数量
type PendingCount struct {
	Count int `json:"count"`
}

// ListApprovals 列出审批
func (h *ApprovalHandler) ListApprovals(c *gin.Context) {
	// status := c.DefaultQuery("status", "pending")
	// page := c.DefaultQuery("page", "1")
	// pageSize := c.DefaultQuery("page_size", "20")

	// 简化实现：返回空列表
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"data":  []Approval{},
			"total": 0,
			"page":  "1",
		},
	})
}

// GetApproval 获取审批详情
func (h *ApprovalHandler) GetApproval(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": Approval{
			ID:        1,
			Type:     "restore",
			Status:   "pending",
			Title:    "恢复审批",
			Content:  "需要恢复数据库",
			Applicant: "admin",
		},
	})
}

// Approve 审批通过
func (h *ApprovalHandler) Approve(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "审批通过"})
}

// Reject 审批拒绝
func (h *ApprovalHandler) Reject(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "审批已拒绝"})
}

// Cancel 取消审批
func (h *ApprovalHandler) Cancel(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已取消"})
}

// PendingCount 获取待审批数量
func (h *ApprovalHandler) PendingCount(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"count": 0,
		},
	})
}
