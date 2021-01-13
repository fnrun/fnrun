// Package http provides a source that is a web server.
package http

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/fnrun/fnrun/pkg/run"
	"github.com/mitchellh/mapstructure"
)

type httpSourceConfig struct {
	Addr              string            `mapstructure:"address,omitempty"`
	TLSKeyFile        string            `mapstructure:"keyFile,omitempty"`
	TLSCertFile       string            `mapstructure:"certFile,omitempty"`
	Base64EncodeBody  bool              `mapstructure:"base64EncodeBody,omitempty"`
	TreatOutputAsBody bool              `mapstructure:"treatOutputAsBody,omitempty"`
	DefaultHeaders    map[string]string `mapstructure:"outputHeaders,omitempty"`
	IgnoreOutput      bool              `mapstructure:"ignoreOutput,omitempty"`
}

type httpSource struct {
	config *httpSourceConfig
}

func (h *httpSource) ConfigureMap(configMap map[string]interface{}) error {
	return mapstructure.Decode(configMap, h.config)
}

func (h *httpSource) Serve(ctx context.Context, f fn.Fn) error {
	errorChan := make(chan error, 1)

	http.HandleFunc("/", makeHandler(ctx, f, h.config))

	srv := &http.Server{Addr: h.config.Addr}

	go func() {
		if h.config.TLSCertFile != "" || h.config.TLSKeyFile != "" {
			errorChan <- srv.ListenAndServeTLS(h.config.TLSCertFile, h.config.TLSKeyFile)
			return
		}
		errorChan <- srv.ListenAndServe()
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}

	return <-errorChan
}

func New() run.Source {
	return &httpSource{
		config: &httpSourceConfig{
			Addr:           ":8080",
			DefaultHeaders: make(map[string]string),
		},
	}
}

func makeHandler(ctx context.Context, f fn.Fn, config *httpSourceConfig) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		input, err := createInput(r, config)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		output, err := f.Invoke(ctx, input)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		if config.TreatOutputAsBody {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprint(output)))
			return
		}

		m, ok := output.(map[string]interface{})
		if !ok {
			log.Printf("expected output to be string or map[string]interface{} but was %T", output)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := writeResponse(w, m, config); err != nil {
			log.Printf("%#v", err)
		}
	}
}

func createInput(r *http.Request, config *httpSourceConfig) (map[string]interface{}, error) {
	input := make(map[string]interface{})

	input["host"] = r.Host
	input["remoteAddress"] = r.RemoteAddr
	input["method"] = r.Method
	input["protocol"] = r.URL.Scheme
	input["contentLength"] = r.ContentLength
	input["url"] = r.URL.String()

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if err = r.Body.Close(); err != nil {
		return nil, err
	}

	if config.Base64EncodeBody {
		input["body"] = base64.StdEncoding.EncodeToString(bodyBytes)
	} else {
		input["body"] = string(bodyBytes)
	}

	cookies := make(map[string]string)
	for _, cookie := range r.Cookies() {
		cookies[cookie.Name] = cookie.Value
	}
	input["cookies"] = cookies

	headers := make(map[string][]string)
	for k, v := range r.Header {
		if k == "Cookie" {
			continue
		}
		headers[k] = v
	}
	input["headers"] = headers

	query := make(map[string][]string)
	for k, v := range r.URL.Query() {
		query[k] = v
	}
	input["query"] = query

	return input, nil
}

type response struct {
	Headers    map[string]string `mapstructure:"headers,omitempty"`
	Body       string            `mapstructure:"body,omitempty"`
	StatusCode int               `mapstructure:"statusCode,omitempty"`
}

func writeResponse(w http.ResponseWriter, m map[string]interface{}, config *httpSourceConfig) error {
	var resp response
	if err := mapstructure.Decode(m, &resp); err != nil {
		return err
	}

	if len(resp.Headers) == 0 {
		for key, value := range config.DefaultHeaders {
			w.Header().Add(key, value)
		}
	}
	for key, value := range resp.Headers {
		w.Header().Add(key, value)
	}

	if resp.StatusCode == 0 {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(resp.StatusCode)
	}

	if !config.IgnoreOutput {
		_, err := w.Write([]byte(resp.Body))
		return err
	}

	return nil
}
