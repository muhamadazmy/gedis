package gedis

import (
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"sync"
)

// LState lua state
type LState struct {
	*lua.LState

	pool *StatePool
}

// Close releases state, and return it back to the pool
func (s *LState) Close() {
	s.SetTop(0)
	s.pool.release(s)
}

// PoolOptions struct
type PoolOptions struct {
	Open func() (*lua.LState, error)
}

var (
	defaultPoolOptions = PoolOptions{
		Open: func() (*lua.LState, error) {
			return lua.NewState(), nil
		},
	}
)

// StatePool represent a pool of lua states
type StatePool struct {
	size uint
	lend uint
	pool []*LState
	opts PoolOptions
	c    *sync.Cond
}

// NewPool creates a new instance of StatePool
func NewPool(size uint, opts ...PoolOptions) *StatePool {
	if size == 0 {
		panic("invalid pool size")
	}
	o := defaultPoolOptions
	if len(opts) != 0 {
		o = opts[0]
	}
	return &StatePool{
		size: size,
		opts: o,
		c:    sync.NewCond(&sync.Mutex{}),
	}
}

func (p *StatePool) String() string {
	return fmt.Sprintf("Size: %d, Available: %d, Borrowed: %d", p.size, len(p.pool), p.lend)
}

// Get or reuse a state from the pool
func (p *StatePool) Get() (*LState, error) {
	p.c.L.Lock()
	defer p.c.L.Unlock()
	var state *LState
	if len(p.pool) > 0 {
		//we have free states
		state = p.pool[0]
		p.pool = p.pool[1:]
		p.lend++
		return state, nil
	}

	// no free states, allocate new one
	for p.lend == p.size {
		p.c.Wait()
	}

	l, err := p.opts.Open()
	if err != nil {
		return nil, err
	}
	state = &LState{
		LState: l,
		pool:   p,
	}

	p.lend++
	return state, nil
}

func (p *StatePool) release(s *LState) {
	p.c.L.Lock()
	defer p.c.L.Unlock()

	p.pool = append(p.pool, s)
	p.lend--
	p.c.Broadcast()
}

// Close the pool, it makes sure that all availabel lua states are closed
// It's okay to reuse the pool after closing. it will reset operation.
func (p *StatePool) Close() error {
	p.c.L.Lock()
	defer p.c.L.Unlock()

	if p.lend > 0 {
		return fmt.Errorf("possible state leakage, not all states has been closed")
	}

	for _, s := range p.pool {
		s.LState.Close()
	}
	p.pool = nil
	return nil
}
