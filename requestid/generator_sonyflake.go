package requestid

import (
	"fmt"
	"strconv"

	"github.com/sony/sonyflake/v2"
)

func UseSnowFlake(st sonyflake.Settings) error {
	sf, err := sonyflake.New(st)
	if err != nil {
		return fmt.Errorf("failed to create sonyflake: %w", err)
	}
	UseGenerator(&sonyFlakeGenerator{sf: sf})

	return nil
}

type sonyFlakeGenerator struct {
	sf *sonyflake.Sonyflake
}

func (g *sonyFlakeGenerator) NextID() (string, error) {
	id, err := g.sf.NextID()
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(id, 10), nil
}
