package app

type AppModule struct {
}

type AppModuleInterface interface {
	Controllers() []interface{}
	Provide() []interface{}
}

func Controllers() []interface{} {
	return []interface{}{}
}

func Provide() []interface{} {
	return []interface{}{}
}
