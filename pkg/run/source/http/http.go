// Package http provides a source that is a web server.
package http

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/fnrun/fnrun/pkg/run"
	"github.com/mitchellh/mapstructure"
)

type httpSource struct {
	Addr                string            `mapstructure:"address,omitempty"`
	TLSKeyFile          string            `mapstructure:"keyFile,omitempty"`
	TLSCertFile         string            `mapstructure:"certFile,omitempty"`
	Base64EncodeBody    bool              `mapstructure:"base64EncodeBody,omitempty"`
	TreatOutputAsBody   bool              `mapstructure:"treatOutputAsBody,omitempty"`
	DefaultHeaders      map[string]string `mapstructure:"outputHeaders,omitempty"`
	IgnoreOutput        bool              `mapstructure:"ignoreOutput,omitempty"`
	ShutdownGracePeriod time.Duration     `mapstructure:"shutdownGracePeriod,omitempty"`
	Listener            net.Listener
}

func (h *httpSource) ConfigureMap(configMap map[string]interface{}) error {
	decoderConfig := &mapstructure.DecoderConfig{
		Metadata:   nil,
		Result:     h,
		DecodeHook: mapstructure.StringToTimeDurationHookFunc(),
	}
	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return err
	}

	err = decoder.Decode(configMap)
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", h.Addr)
	if err != nil {
		return err
	}

	h.Listener = ln
	return nil
}

func (h *httpSource) Serve(ctx context.Context, f fn.Fn) error {
	errorChan := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/", h.makeHandler(ctx, f))

	srv := &http.Server{
		Addr:    h.Addr,
		Handler: mux,
	}

	go func() {
		if h.TLSCertFile != "" || h.TLSKeyFile != "" {
			errorChan <- srv.ServeTLS(h.Listener, h.TLSCertFile, h.TLSKeyFile)
			return
		}
		errorChan <- srv.Serve(h.Listener)
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), h.ShutdownGracePeriod)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}

	return <-errorChan
}

func (h *httpSource) makeHandler(ctx context.Context, f fn.Fn) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		input, err := h.createInput(r)
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

		if h.TreatOutputAsBody {
			if err := h.writeResponse(w, map[string]interface{}{"body": fmt.Sprint(output)}); err != nil {
				log.Printf("%#v", err)
			}
			return
		}

		m, ok := output.(map[string]interface{})
		if !ok {
			log.Printf("expected output to be map[string]interface{} but was %T", output)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := h.writeResponse(w, m); err != nil {
			log.Printf("%#v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

func (h *httpSource) createInput(r *http.Request) (map[string]interface{}, error) {
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
	defer r.Body.Close()

	if h.Base64EncodeBody {
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

func (h *httpSource) writeResponse(w http.ResponseWriter, m map[string]interface{}) error {
	resp := &struct {
		Headers    map[string]string `mapstructure:"headers,omitempty"`
		Body       string            `mapstructure:"body,omitempty"`
		StatusCode int               `mapstructure:"statusCode,omitempty"`
	}{}

	if err := mapstructure.Decode(m, &resp); err != nil {
		return err
	}

	if len(resp.Headers) == 0 {
		for key, value := range h.DefaultHeaders {
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

	if !h.IgnoreOutput {
		_, err := w.Write([]byte(resp.Body))
		return err
	}

	return nil
}

// New returns a new source that with default values. When Serve is called on
// the resulting object, the source will start a new HTTP server based on its
// configuration and invoke a function with values received as HTTP requests.
func New() run.Source {
	return &httpSource{
		Addr:                ":8080",
		DefaultHeaders:      make(map[string]string),
		ShutdownGracePeriod: 10 * time.Second,
	}
}
