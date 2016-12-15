// Package pool is a generic, high-performance pool for net.Conn
// objects.
package pool

import (
	"net"
	"sync/atomic"
	"time"
	"unsafe"
)

// Factory must returns new connections
type Factory func() (net.Conn, error)

// Options can tweak Pool configuration
type Options struct {
	// InitialSize creates a number of connection on pool initialization
	// Default: 0
	InitialSize int

	// MaxCap sets the maximum pool capacity. Will be automatically adjusted when InitialSize
	// is larger.
	// Default: 10
	MaxCap int

	// IdleTimeout timeout after which connections are reaped and
	// automatically removed from the pool.
	// Default: 0 (= never)
	IdleTimeout time.Duration

	// ReapInterval determines the frequency of reap cycles
	// Default: 1 minute
	ReapInterval time.Duration
}

func (o *Options) norm() Options {
	x := *o
	if x.ReapInterval <= 0 {
		x.ReapInterval = time.Minute
	}
	if x.MaxCap <= 0 {
		x.MaxCap = 10
	}
	if x.MaxCap < x.InitialSize {
		x.MaxCap = x.InitialSize
	}
	return x
}

type none struct{}

// Pool contains a number of connections
type Pool struct {
	head    unsafe.Pointer
	opt     Options
	factory Factory

	dying, dead chan none

	avail  int32
	closed int32
}

// New creates a pool with an initial number of connection and a maximum cap
func New(opt *Options, factory Factory) (*Pool, error) {
	if opt == nil {
		opt = new(Options)
	}

	p := &Pool{
		factory: factory,
		opt:     opt.norm(),
		dying:   make(chan none),
		dead:    make(chan none),
	}

	for i := 0; i < opt.InitialSize; i++ {
		cn, err := factory()
		if err != nil {
			_ = p.close()
			return nil, err
		}
		p.Put(cn)
	}

	go p.loop()
	return p, nil
}

// Len returns the number of available connections in the pool
func (s *Pool) Len() int { return int(atomic.LoadInt32(&s.avail)) }

// Get returns a connection from the pool or creates a new one
func (s *Pool) Get() (net.Conn, error) {
	if cn := s.pop(); cn != nil {
		return cn, nil
	}

	return s.factory()
}

// Put adds/returns a connection to the pool
func (s *Pool) Put(cn net.Conn) bool {
	if s.Len() >= s.opt.MaxCap || atomic.LoadInt32(&s.closed) == 1 {
		_ = cn.Close()
		return false
	}

	m := &poolMember{
		Conn:       cn,
		lastAccess: time.Now(),
	}
	for {
		m.next = atomic.LoadPointer(&s.head)
		if atomic.CompareAndSwapPointer(&s.head, m.next, unsafe.Pointer(m)) {
			atomic.AddInt32(&s.avail, 1)
			return true
		}
	}
}

// Close closes all connections and the pool
func (s *Pool) Close() error {
	if !atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		return nil
	}

	close(s.dying)
	<-s.dead
	return s.close()
}

func (s *Pool) pop() net.Conn {
	for {
		head := atomic.LoadPointer(&s.head)
		if head == nil {
			return nil
		}
		if atomic.CompareAndSwapPointer(&s.head, head, (*poolMember)(head).next) {
			atomic.AddInt32(&s.avail, -1)
			return (*poolMember)(head).Conn
		}
	}
}

func (s *Pool) close() (err error) {
	for {
		cn := s.pop()
		if cn == nil {
			break
		}
		if e := cn.Close(); e != nil {
			err = e
		}
	}
	return err
}

func (s *Pool) reap() {
	timeout := s.opt.IdleTimeout
	if timeout <= 0 {
		return
	}
}

func (s *Pool) loop() {
	defer close(s.dead)

	ticker := time.NewTicker(s.opt.ReapInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.dying:
			return
		case <-ticker.C:
			s.reap()
		}
	}
}

type poolMember struct {
	net.Conn
	next       unsafe.Pointer
	lastAccess time.Time
}
