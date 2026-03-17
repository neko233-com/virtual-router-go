package rpc_test

import (
	"testing"

	"github.com/neko233-com/virtual-router-go/internal/rpc"
)

func TestFutureManagerSuccess(t *testing.T) {
	fm := rpc.NewFutureManager()
	f := rpc.NewFuture("uid-1")
	fm.Register(f)

	fm.SetSuccess("uid-1", "ok")
	res, err := f.Await(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != "ok" {
		t.Fatalf("unexpected result: %s", res)
	}
}

func TestFutureManagerError(t *testing.T) {
	fm := rpc.NewFutureManager()
	f := rpc.NewFuture("uid-2")
	fm.Register(f)

	fm.SetError("uid-2", "boom")
	_, err := f.Await(0)
	if err == nil {
		t.Fatalf("expected error")
	}
}
