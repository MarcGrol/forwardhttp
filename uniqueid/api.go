package uniqueid

//go:generate mockgen -source=api.go -destination=gen_GeneratorMock.go -package=uniqueid github.com/MarcGrol/forwardhttp/uniqueid Generator

type Generator interface {
	Generate() string
}
