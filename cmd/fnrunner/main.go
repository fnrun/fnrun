package main

import (
	"context"
	"flag"
	"io/ioutil"
	"os"

	"github.com/fnrun/fnrun/run"
	"github.com/fnrun/fnrun/run/config"
	"github.com/fnrun/fnrun/run/fn/cli"
	httpfn "github.com/fnrun/fnrun/run/fn/http"
	"github.com/fnrun/fnrun/run/fn/identity"
	fnloader "github.com/fnrun/fnrun/run/fn/loader"
	"github.com/fnrun/fnrun/run/fn/pool"
	"github.com/fnrun/fnrun/run/middleware/circuitbreaker"
	"github.com/fnrun/fnrun/run/middleware/debug"
	"github.com/fnrun/fnrun/run/middleware/jq"
	"github.com/fnrun/fnrun/run/middleware/json"
	kafkamiddleware "github.com/fnrun/fnrun/run/middleware/kafka"
	"github.com/fnrun/fnrun/run/middleware/key"
	"github.com/fnrun/fnrun/run/middleware/pipeline"
	"github.com/fnrun/fnrun/run/middleware/ratelimiter"
	"github.com/fnrun/fnrun/run/middleware/timeout"
	"github.com/fnrun/fnrun/run/runner"
	"github.com/fnrun/fnrun/run/source/cron"
	"github.com/fnrun/fnrun/run/source/http"
	"github.com/fnrun/fnrun/run/source/kafka"
	"github.com/fnrun/fnrun/run/source/lambda"
	sourceloader "github.com/fnrun/fnrun/run/source/loader"
	"github.com/fnrun/fnrun/run/source/sqs"
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
	registry.RegisterFn("fnrun.fn/http", httpfn.New)
	registry.RegisterFn("fnrun.fn/identity", identity.New)
	registry.RegisterFnWithRegistry("fnrun.fn/pool", pool.New)
	registry.RegisterFnWithRegistry("fn", fnloader.New)

	registry.RegisterMiddleware("fnrun.middleware/circuitbreaker", circuitbreaker.New)
	registry.RegisterMiddleware("fnrun.middleware/debug", debug.New)
	registry.RegisterMiddleware("fnrun.middleware/jq", jq.New)
	registry.RegisterMiddleware("fnrun.middleware/json", json.New)
	registry.RegisterMiddleware("fnrun.middleware/kafka", kafkamiddleware.New)
	registry.RegisterMiddleware("fnrun.middleware/key", key.New)
	registry.RegisterMiddleware("fnrun.middleware/ratelimiter", ratelimiter.New)
	registry.RegisterMiddleware("fnrun.middleware/timeout", timeout.New)
	registry.RegisterMiddlewareWithRegistry("middleware", pipeline.NewWithRegistry)

	registry.RegisterSource("fnrun.source/cron", cron.New)
	registry.RegisterSource("fnrun.source/http", http.New)
	registry.RegisterSource("fnrun.source/kafka", kafka.New)
	registry.RegisterSource("fnrun.source/lambda", lambda.New)
	registry.RegisterSource("fnrun.source/sqs", sqs.New)
	registry.RegisterSourceWithRegistry("source", sourceloader.New)

	configBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	configStr := os.ExpandEnv(string(configBytes))

	var configMap map[string]interface{}
	err = yaml.Unmarshal([]byte(configStr), &configMap)
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
