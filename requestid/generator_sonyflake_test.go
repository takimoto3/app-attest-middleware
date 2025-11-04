package requestid

import (
	"sync/atomic"
	"testing"

	"github.com/sony/sonyflake/v2"
)

func TestUseSnowFlake(t *testing.T) {
	t.Cleanup(func() {
		generator = atomic.Value{}
	})

	sfSettings := sonyflake.Settings{}
	err := UseSnowFlake(sfSettings)
	if err != nil {
		t.Fatalf("UseSnowFlake failed: %v", err)
	}

	gen := currentGenerator()
	if gen == nil {
		t.Fatal("generator not initialized after UseSnowFlake")
	}

	id, err := gen.NextID()
	if err != nil {
		t.Fatalf("NextID failed: %v", err)
	}
	if id == "" {
		t.Error("NextID returned empty string")
	}
}
