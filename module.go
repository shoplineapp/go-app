package app

type AppModule struct {
}

type AppModuleInterface interface {
	Provide() []interface{}
}

func Provide() []interface{} {
	return []interface{}{}
}
