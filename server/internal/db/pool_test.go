package db

import (
	"testing"
)

func TestDefaultPoolConfig(t *testing.T) {
	cfg := DefaultPoolConfig()
	if cfg.MaxConns <= 0 {
		t.Errorf("expected positive MaxConns, got %d", cfg.MaxConns)
	}
	if cfg.MinConns < 0 {
		t.Errorf("expected non-negative MinConns, got %d", cfg.MinConns)
	}
}
