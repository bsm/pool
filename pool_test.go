package pool_test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bsm/pool"
)

func TestPool(t *testing.T) {
	server, factory := mockServer()
	defer server.Close()

	pool, err := pool.New(&pool.Options{
		InitialSize: 3,
		MaxCap:      5,
	}, factory)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer pool.Close()

	// there should be 3 active connections in the pool
	if exp, got := 3, pool.Len(); err != nil {
		t.Errorf("expected %v, got %v", exp, got)
	}

	// check out 6 connections
	var cns []net.Conn
	for i := 0; i < 6; i++ {
		cn, err := pool.Get()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		cns = append(cns, cn)
	}

	for _, cn := range cns[:5] {
		if !pool.Put(cn) {
			t.Error("expected true")
		}
	}
	if exp, got := 5, pool.Len(); err != nil {
		t.Errorf("expected %v, got %v", exp, got)
	}

	for _, cn := range cns[5:] {
		if pool.Put(cn) {
			t.Error("expected false")
		}
	}
	if exp, got := 5, pool.Len(); err != nil {
		t.Errorf("expected %v, got %v", exp, got)
	}
}

// --------------------------------------------------------------------

func mockServer() (*httptest.Server, pool.Factory) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	factory := func() (net.Conn, error) {
		return net.Dial("tcp", strings.Replace(server.URL, "http://", "", -1))
	}
	return server, factory
}

func BenchmarkPool(b *testing.B) {
	srv, factory := mockServer()
	defer srv.Close()

	p, err := pool.New(nil, factory)
	if err != nil {
		b.Fatal(err)
	}
	defer p.Close()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cn, err := p.Get()
			if err != nil {
				b.Fatal(err)
			}
			p.Put(cn)
		}
	})
}
