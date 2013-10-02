package main

import (
	"testing"
)

func Test_PoolReserve(t *testing.T) {
	p := NewUidPool(10, 11)

	uid, _ := p.Reserve()
	if uid != 10 {
		t.Errorf("First uid was not %d, not 10", uid)
	}

	uid, _ = p.Reserve()
	if uid != 11 {
		t.Errorf("Second uid was not %d, not 11", uid)
	}
}

func Test_PoolRelease(t *testing.T) {
	p := NewUidPool(10, 11)

	p.Reserve()
	p.Reserve()

	if p.Size != 0 {
		t.Error("Size should be 0")
	}

	p.Release(11)
	p.Release(10)

	uid, _ := p.Reserve()
	if uid != 11 {
		t.Error("First uid was not 11")
	}

	uid, _ = p.Reserve()
	if uid != 10 {
		t.Error("Second uid was not 10")
	}
}

func Test_PoolEmpty(t *testing.T) {
	p := NewUidPool(10, 10)

	// Use up only uid
	p.Reserve()

	// Try to reserve from empty pool
	_, err := p.Reserve()
	if err == nil {
		t.Error("Managed to reserve from empty pool")
	}
}

func Test_PoolFull(t *testing.T) {
	p := NewUidPool(10, 10)

	// Try to reserve from empty pool
	err := p.Release(11)
	if err == nil {
		t.Error("Managed to release to a full pool")
	}
}
