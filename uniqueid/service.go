package uniqueid

import (
	"strings"

	"github.com/google/uuid"
)

type generator struct{}

func NewGenerator() Generator {
	return &generator{}
}

func (_ generator) Generate() string {
	id, _ := uuid.NewUUID()
	// gcloud cloud dos not like the minus character
	return strings.Replace(id.String(), "-", "", -1)
}
