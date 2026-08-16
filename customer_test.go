package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func testRecord(name, phone, patient, city string) CustomerImportRecord {
	return CustomerImportRecord{Name: name, Phone: phone, PatientName: patient, Relationship: "本人", ServiceCity: city, FollowUpAt: "2026-09-01T09:00:00+08:00", Note: "需要轮椅"}
}

func TestCustomerLifecycleAndOperationLogs(t *testing.T) {
	service := NewCustomerService(NewMemoryCustomerRepository(), NewMemoryOperationLogRepository())
	created, err := service.Create(testRecord("李梅", "13800000001", "李梅", "上海").customer(""))
	if err != nil {
		t.Fatal(err)
	}
	created.Note = "已确认上午陪诊"
	if _, err := service.Update(created.ID, created); err != nil {
		t.Fatal(err)
	}
	if len(service.List()) != 1 || service.List()[0].Note != "已确认上午陪诊" {
		t.Fatalf("unexpected customers: %#v", service.List())
	}
	logs := service.Logs()
	if len(logs) != 2 || logs[0].Action != "customer.create" || logs[1].Action != "customer.update" {
		t.Fatalf("unexpected logs: %#v", logs)
	}
}

func TestImportPreviewReportsDuplicatePhonesWithoutWriting(t *testing.T) {
	defer failOnPanic(t)()
	service := NewCustomerService(NewMemoryCustomerRepository(), NewMemoryOperationLogRepository())
	records, _ := importFixture(t, "testdata/import_duplicate_phone.json")
	preview := service.PreviewImport(records)
	if len(preview.DuplicatePhones) != 1 || preview.DuplicatePhones[0] != "13900000002" {
		t.Fatalf("unexpected duplicate phones: %#v", preview.DuplicatePhones)
	}
	if len(service.List()) != 0 {
		t.Fatalf("preview wrote customers: %#v", service.List())
	}
	logs := service.Logs()
	if len(logs) != 1 || logs[0].Action != "customer.import.preview" {
		t.Fatalf("unexpected logs: %#v", logs)
	}
}

func TestCommitImportRejectsExistingPhone(t *testing.T) {
	defer failOnPanic(t)()
	service := NewCustomerService(NewMemoryCustomerRepository(), NewMemoryOperationLogRepository())
	if _, err := service.Create(testRecord("王芳", "13700000003", "王芳", "北京").customer("")); err != nil {
		t.Fatal(err)
	}
	_, err := service.CommitImport([]CustomerImportRecord{testRecord("另一位", "13700000003", "王芳", "北京")})
	if !errors.Is(err, ErrDuplicatePhone) {
		t.Fatalf("expected duplicate phone error, got %v", err)
	}
	if len(service.List()) != 1 {
		t.Fatalf("commit changed customers: %#v", service.List())
	}
}

func TestHTTPWorkflow(t *testing.T) {
	defer failOnPanic(t)()
	service := NewCustomerService(NewMemoryCustomerRepository(), NewMemoryOperationLogRepository())
	handler := NewHTTPServer(service)
	_, body := importFixture(t, "testdata/import_valid.json")
	req := httptest.NewRequest(http.MethodPost, "/api/customers/import", strings.NewReader(string(body)))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
	logsReq := httptest.NewRequest(http.MethodGet, "/api/operation-logs", nil)
	logsRec := httptest.NewRecorder()
	handler.ServeHTTP(logsRec, logsReq)
	if logsRec.Code != http.StatusOK || !strings.Contains(logsRec.Body.String(), "customer.import") {
		t.Fatalf("unexpected logs response: %d %s", logsRec.Code, logsRec.Body.String())
	}
}

func TestHTTPStaticAssets(t *testing.T) {
	handler := NewHTTPServer(NewCustomerService(NewMemoryCustomerRepository(), NewMemoryOperationLogRepository()))
	for _, path := range []string{"/", "/assets/styles.css", "/assets/app.js"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status for %s: %d", path, rec.Code)
		}
	}
}

func importFixture(t *testing.T, path string) ([]CustomerImportRecord, []byte) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var records []CustomerImportRecord
	if err := json.Unmarshal(body, &records); err != nil {
		t.Fatal(err)
	}
	return records, body
}

func failOnPanic(t *testing.T) func() {
	t.Helper()
	return func() {
		if value := recover(); value != nil {
			t.Errorf("workflow panicked: %v", value)
		}
	}
}
