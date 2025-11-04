package requestid

import "github.com/google/uuid"

// UseUUID sets the global request ID generator to use UUID v4 (default).
func UseUUID() {
	UseGenerator(&uuidV4Generator{})
}

// UseUUIDv6 sets the generator to use UUID version 6 (time-ordered).
func UseUUIDv6() {
	UseGenerator(&uuidV6Generator{})
}

// uuidGenerator implements the Generator interface using UUID v4.
type uuidV4Generator struct{}

// NextID generates a new UUID v4 string.
func (g *uuidV4Generator) NextID() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

// uuidGenerator implements the Generator interface using UUID v6.
type uuidV6Generator struct{}

// NextID generates a new UUID v6 string.
func (g *uuidV6Generator) NextID() (string, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}
