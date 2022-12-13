package env

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/shoplineapp/go-app/plugins"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewEnv)
}

var env *Env

type Env struct {
	defaultValues map[string]string
}

func (e *Env) SetDefaultEnv(values map[string]string) {
	for key, value := range values {
		e.defaultValues[key] = value
	}
}

func (e Env) GetEnv(key string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return e.defaultValues[key]
}

func (e Env) GetEnvInt(key string) int {
	val := e.GetEnv(key)
	intVal, _ := strconv.Atoi(val)
	return intVal
}

func NewEnv() *Env {
	env = &Env{
		defaultValues: map[string]string{
			"ENVIRONMENT": "development",
		},
	}

	projectRoot := os.Getenv("PROJECT_ROOT")
	if len(projectRoot) == 0 {
		projectRoot, _ = os.Getwd()
	}

	path := fmt.Sprintf("%s/.env", projectRoot)

	if os.Getenv("ENVIRONMENT") == "test" {
		path = fmt.Sprintf("%s.test", path)
	}

	if err := godotenv.Load(path); err != nil {
		log.Print("No .env file found", err)
	}

	return env
}
