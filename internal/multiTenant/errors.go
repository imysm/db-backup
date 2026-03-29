package multiTenant

import "errors"

var (
	// ErrQuotaExceeded 配额超限
	ErrQuotaExceeded = errors.New("quota exceeded")

	// ErrTenantNotFound 租户不存在
	ErrTenantNotFound = errors.New("tenant not found")

	// ErrTenantDisabled 租户已禁用
	ErrTenantDisabled = errors.New("tenant is disabled")

	// ErrTenantExpired 租户已过期
	ErrTenantExpired = errors.New("tenant expired")

	// ErrFeatureNotAllowed 功能不允许
	ErrFeatureNotAllowed = errors.New("feature not allowed for current plan")
)
