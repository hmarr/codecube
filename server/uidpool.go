package main

import (
	"sync"
	"errors"
)

// FIFO UID pool
type UidPool struct {
	pool    []int
	rIdx    int
	lIdx    int
	Size    int
	sync.Mutex
}

func NewUidPool(lower int, upper int) *UidPool {
	size := upper - lower + 1
	pool := make([]int, size)
	for i := 0; i < size; i += 1 {
		pool[i] = lower + i
	}
	return &UidPool{
		pool: pool,
		lIdx: 0,
		rIdx: 0,
		Size: size,
	}
}

func (p *UidPool) Reserve() (int, error) {
	p.Lock()
	defer p.Unlock()

	if p.Size == 0 {
		return 0, errors.New("Uid pool is empty")
	}

	uid := p.pool[p.rIdx]
	p.pool[p.rIdx] = -1
	p.Size -= 1
	p.rIdx = p.idxSucc(p.rIdx)
	return uid, nil
}

func (p *UidPool) Release(uid int) (error) {
	p.Lock()
	defer p.Unlock()

	if p.Size == cap(p.pool) {
		return errors.New("Uid pool is full")
	}

	p.pool[p.lIdx] = uid
	p.Size += 1
	p.lIdx = p.idxSucc(p.lIdx)
	return nil
}

func (p *UidPool) idxSucc(i int) int {
	return (i + 1) % cap(p.pool)
}

