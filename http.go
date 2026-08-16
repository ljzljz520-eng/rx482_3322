package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type HTTPServer struct {
	service *CustomerService
}

func NewHTTPServer(service *CustomerService) http.Handler {
	server := &HTTPServer{service: service}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/customers", server.listCustomers)
	mux.HandleFunc("POST /api/customers", server.createCustomer)
	mux.HandleFunc("PUT /api/customers/", server.updateCustomer)
	mux.HandleFunc("POST /api/customers/import/preview", server.previewImport)
	mux.HandleFunc("POST /api/customers/import", server.commitImport)
	mux.HandleFunc("GET /api/operation-logs", server.listLogs)
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("web"))))
	mux.HandleFunc("GET /", server.index)
	return mux
}

func (s *HTTPServer) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "web/index.html")
}

func (s *HTTPServer) listCustomers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.service.List())
}

func (s *HTTPServer) createCustomer(w http.ResponseWriter, r *http.Request) {
	var customer Customer
	if !decodeJSON(w, r, &customer) {
		return
	}
	created, err := s.service.Create(customer)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *HTTPServer) updateCustomer(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/customers/")
	if id == "" || strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}
	var customer Customer
	if !decodeJSON(w, r, &customer) {
		return
	}
	updated, err := s.service.Update(id, customer)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *HTTPServer) previewImport(w http.ResponseWriter, r *http.Request) {
	var records []CustomerImportRecord
	if !decodeJSON(w, r, &records) {
		return
	}
	writeJSON(w, http.StatusOK, s.service.PreviewImport(records))
}

func (s *HTTPServer) commitImport(w http.ResponseWriter, r *http.Request) {
	var records []CustomerImportRecord
	if !decodeJSON(w, r, &records) {
		return
	}
	created, err := s.service.CommitImport(records)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *HTTPServer) listLogs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.service.Logs())
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(target); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return false
	}
	return true
}

func writeServiceError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, ErrCustomerNotFound) {
		status = http.StatusNotFound
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
