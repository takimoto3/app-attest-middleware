package requestid

import (
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
)

func TestUseUUID(t *testing.T) {
	t.Cleanup(func() {
		generator = atomic.Value{}
	})

	UseUUID() // Initialize UUIDv4 generator

	gen := currentGenerator()
	if gen == nil {
		t.Fatal("generator not initialized after UseUUID")
	}

	id, err := gen.NextID()
	if err != nil {
		t.Fatalf("NextID failed: %v", err)
	}
	if id == "" {
		t.Error("NextID returned empty string")
	}
	// Validate UUID format (optional but useful)
	if _, err := uuid.Parse(id); err != nil {
		t.Errorf("invalid UUID format: %v", err)
	}
}

func TestUseUUIDv6(t *testing.T) {
	t.Cleanup(func() {
		generator = atomic.Value{}
	})

	UseUUIDv6() // Initialize UUIDv6 generator

	gen := currentGenerator()
	if gen == nil {
		t.Fatal("generator not initialized after UseUUIDv6")
	}

	id, err := gen.NextID()
	if err != nil {
		t.Fatalf("NextID failed: %v", err)
	}
	if id == "" {
		t.Error("NextID returned empty string")
	}
	// Validate UUID format (optional but useful)
	if _, err := uuid.Parse(id); err != nil {
		t.Errorf("invalid UUIDv6 format: %v", err)
	}
}
