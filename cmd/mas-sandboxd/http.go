package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/voocel/mas/executor/sandbox"
	"github.com/voocel/mas/executor/sandbox/manager"
	"github.com/voocel/mas/executor/sandbox/policy"
	"github.com/voocel/mas/executor/sandbox/runtime"
)

const maxRequestBytes = 1 << 20

func runHTTP(listen, authToken string, rt runtime.Runtime, evaluator policy.Evaluator) error {
	if strings.TrimSpace(listen) == "" {
		return errors.New("listen address is empty")
	}
	if rt == nil {
		return errors.New("runtime is nil")
	}

	svc := &manager.Service{Runtime: rt, Evaluator: evaluator}

	server := &http.Server{
		Addr:         listen,
		Handler:      newHTTPHandler(authToken, svc),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	fmt.Println("sandbox http listening on", listen)
	return server.ListenAndServe()
}

func newHTTPHandler(authToken string, svc *manager.Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sandbox/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := authorize(r, authToken); err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		resp, err := svc.Health(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, resp)
	})
	mux.HandleFunc("/v1/sandbox/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := authorize(r, authToken); err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var req sandbox.CreateSandboxRequest
		if err := decodeJSON(w, r, &req); err != nil {
			return
		}
		resp, err := svc.CreateSandbox(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, resp)
	})
	mux.HandleFunc("/v1/sandbox/execute", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := authorize(r, authToken); err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var req sandbox.ExecuteToolRequest
		if err := decodeJSON(w, r, &req); err != nil {
			return
		}
		resp, err := svc.ExecuteTool(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, resp)
	})
	mux.HandleFunc("/v1/sandbox/destroy", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := authorize(r, authToken); err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var req sandbox.DestroySandboxRequest
		if err := decodeJSON(w, r, &req); err != nil {
			return
		}
		resp, err := svc.DestroySandbox(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, resp)
	})
	return mux
}

func decodeJSON(w http.ResponseWriter, r *http.Request, out any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return err
	}
	if len(data) == 0 {
		http.Error(w, "empty request body", http.StatusBadRequest)
		return errors.New("empty request body")
	}
	if err := json.Unmarshal(data, out); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	_ = enc.Encode(payload)
}

func authorize(r *http.Request, token string) error {
	if strings.TrimSpace(token) == "" {
		return nil
	}
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		if strings.TrimSpace(header[7:]) == token {
			return nil
		}
	}
	if strings.TrimSpace(r.Header.Get("X-Sandbox-Token")) == token {
		return nil
	}
	return errors.New("unauthorized")
}
