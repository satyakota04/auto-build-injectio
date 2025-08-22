package autodetect

type Injecter interface {
	InjectTool() error
}
