package main

import (
	"context"
	"flag"
	"io/ioutil"
	"os"

	"github.com/fnrun/fnrun/pkg/run"
	"github.com/fnrun/fnrun/pkg/run/config"
	"github.com/fnrun/fnrun/pkg/run/fn/cli"
	"github.com/fnrun/fnrun/pkg/run/fn/identity"
	fnloader "github.com/fnrun/fnrun/pkg/run/fn/loader"
	"github.com/fnrun/fnrun/pkg/run/fn/pool"
	"github.com/fnrun/fnrun/pkg/run/middleware/debug"
	"github.com/fnrun/fnrun/pkg/run/middleware/json"
	"github.com/fnrun/fnrun/pkg/run/middleware/key"
	"github.com/fnrun/fnrun/pkg/run/middleware/pipeline"
	"github.com/fnrun/fnrun/pkg/run/middleware/timeout"
	"github.com/fnrun/fnrun/pkg/run/runner"
	"github.com/fnrun/fnrun/pkg/run/source/http"
	"github.com/fnrun/fnrun/pkg/run/source/kafka"
	"github.com/fnrun/fnrun/pkg/run/source/lambda"
	sourceloader "github.com/fnrun/fnrun/pkg/run/source/loader"
	"github.com/fnrun/fnrun/pkg/run/source/sqs"
	"gopkg.in/yaml.v3"
)

func main() {
	var filePath string
	flag.StringVar(&filePath, "f", "fnrun.yaml", "path to configuration yaml file")
	flag.Parse()

	if envFilePath := os.Getenv("CONFIG_FILE"); envFilePath != "" {
		filePath = envFilePath
	}

	registry := run.NewRegistry()

	registry.RegisterFn("fnrun.fn/cli", cli.New)
	registry.RegisterFn("fnrun.fn/identity", identity.New)
	registry.RegisterFnWithRegistry("fnrun.fn/pool", pool.New)
	registry.RegisterFnWithRegistry("fn", fnloader.New)

	registry.RegisterMiddleware("fnrun.middleware/debug", debug.New)
	registry.RegisterMiddleware("fnrun.middleware/json", json.New)
	registry.RegisterMiddleware("fnrun.middleware/key", key.New)
	registry.RegisterMiddleware("fnrun.middleware/timeout", timeout.New)
	registry.RegisterMiddlewareWithRegistry("middleware", pipeline.NewWithRegistry)

	registry.RegisterSource("fnrun.source/http", http.New)
	registry.RegisterSource("fnrun.source/kafka", kafka.New)
	registry.RegisterSource("fnrun.source/lambda", lambda.New)
	registry.RegisterSource("fnrun.source/sqs", sqs.New)
	registry.RegisterSourceWithRegistry("source", sourceloader.New)

	configBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	var configMap map[string]interface{}
	err = yaml.Unmarshal(configBytes, &configMap)
	if err != nil {
		panic(err)
	}

	runner := runner.New(registry)
	err = config.Configure(runner, configMap)
	if err != nil {
		panic(err)
	}

	if err = runner.Run(context.Background()); err != nil {
		panic(err)
	}
}
