package service

import (
	"errors"
	"net"
	"testing"
)

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"connection refused", errors.New("dial tcp: connection refused"), true},
		{"deadline exceeded", errors.New("context deadline exceeded"), true},
		{"normal error", errors.New("invalid config"), false},
		{"random string", errors.New("some random error"), false},
		{"net.Error timeout", &netErrTimeout{}, true},
		{"net.Error temporary", &netErrTemporary{}, false},
		{"contains network", errors.New("network is unreachable"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRetryableError(tt.err)
			if got != tt.want {
				t.Errorf("isRetryableError() = %v, want %v", got, tt.want)
			}
		})
	}
}

type netErrTimeout struct{}

func (e *netErrTimeout) Error() string   { return "timeout" }
func (e *netErrTimeout) Timeout() bool   { return true }
func (e *netErrTimeout) Temporary() bool { return false }

type netErrTemporary struct{}

func (e *netErrTemporary) Error() string   { return "temporary" }
func (e *netErrTemporary) Timeout() bool   { return false }
func (e *netErrTemporary) Temporary() bool { return true }

var _ net.Error = (*netErrTimeout)(nil)
var _ net.Error = (*netErrTemporary)(nil)
