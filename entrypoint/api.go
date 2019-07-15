package entrypoint

import (
	"github.com/MarcGrol/forwardhttp/forwarder"
	"github.com/MarcGrol/forwardhttp/uniqueid"
)

type webService struct {
	uidGenerator uniqueid.Generator
	forwarder    forwarder.Forwarder
}
