package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

var (
	ErrCustomerNotFound = errors.New("customer not found")
	ErrDuplicatePhone   = errors.New("duplicate phone number")
	ErrInvalidCustomer  = errors.New("invalid customer")
)

type Customer struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Phone        string `json:"phone"`
	PatientName  string `json:"patient_name"`
	Relationship string `json:"relationship"`
	ServiceCity  string `json:"service_city"`
	FollowUpAt   string `json:"follow_up_at"`
	Note         string `json:"note"`
}

type CustomerImportRecord struct {
	Name         string `json:"name"`
	Phone        string `json:"phone"`
	PatientName  string `json:"patient_name"`
	Relationship string `json:"relationship"`
	ServiceCity  string `json:"service_city"`
	FollowUpAt   string `json:"follow_up_at"`
	Note         string `json:"note"`
}

func (r CustomerImportRecord) customer(id string) Customer {
	return Customer{ID: id, Name: strings.TrimSpace(r.Name), Phone: strings.TrimSpace(r.Phone), PatientName: strings.TrimSpace(r.PatientName), Relationship: strings.TrimSpace(r.Relationship), ServiceCity: strings.TrimSpace(r.ServiceCity), FollowUpAt: strings.TrimSpace(r.FollowUpAt), Note: strings.TrimSpace(r.Note)}
}

type CustomerRepository interface {
	List() []Customer
	Get(id string) (Customer, error)
	Create(customer Customer) (Customer, error)
	Put(customer Customer) error
}

type MemoryCustomerRepository struct {
	customers map[string]Customer
	mu        sync.RWMutex
	sequence  int
}

func NewMemoryCustomerRepository() *MemoryCustomerRepository {
	return &MemoryCustomerRepository{customers: make(map[string]Customer)}
}

func (r *MemoryCustomerRepository) List() []Customer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	customers := make([]Customer, 0, len(r.customers))
	for _, customer := range r.customers {
		customers = append(customers, customer)
	}
	sort.Slice(customers, func(i, j int) bool { return customers[i].ID < customers[j].ID })
	return customers
}

func (r *MemoryCustomerRepository) Get(id string) (Customer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	customer, ok := r.customers[id]
	if !ok {
		return Customer{}, ErrCustomerNotFound
	}
	return customer, nil
}

func (r *MemoryCustomerRepository) Create(customer Customer) (Customer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.customers {
		if existing.Phone == customer.Phone {
			return Customer{}, ErrDuplicatePhone
		}
	}
	r.sequence++
	customer.ID = fmt.Sprintf("C%04d", r.sequence)
	r.customers[customer.ID] = customer
	return customer, nil
}

func (r *MemoryCustomerRepository) Put(customer Customer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if customer.ID == "" {
		return ErrInvalidCustomer
	}
	for id, existing := range r.customers {
		if id != customer.ID && existing.Phone == customer.Phone {
			return ErrDuplicatePhone
		}
	}
	r.customers[customer.ID] = customer
	return nil
}

type OperationLog struct {
	ID     int    `json:"id"`
	Action string `json:"action"`
	Target string `json:"target"`
	Detail string `json:"detail"`
	At     string `json:"at"`
}

type OperationLogRepository interface {
	Append(log OperationLog) OperationLog
	List() []OperationLog
}

type MemoryOperationLogRepository struct {
	logs []OperationLog
	mu   sync.RWMutex
}

func NewMemoryOperationLogRepository() *MemoryOperationLogRepository {
	return &MemoryOperationLogRepository{}
}

func (r *MemoryOperationLogRepository) Append(entry OperationLog) OperationLog {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry.ID = len(r.logs) + 1
	if entry.At == "" {
		entry.At = fmt.Sprintf("operation-%04d", entry.ID)
	}
	r.logs = append(r.logs, entry)
	return entry
}

func (r *MemoryOperationLogRepository) List() []OperationLog {
	r.mu.RLock()
	defer r.mu.RUnlock()
	logs := make([]OperationLog, len(r.logs))
	copy(logs, r.logs)
	return logs
}

type ImportPreview struct {
	Records         []CustomerImportRecord `json:"records"`
	DuplicatePhones []string               `json:"duplicate_phones"`
	ExistingPhones  []string               `json:"existing_phones"`
}

type CustomerService struct {
	customers CustomerRepository
	logs      OperationLogRepository
}

func NewCustomerService(customers CustomerRepository, logs OperationLogRepository) *CustomerService {
	return &CustomerService{customers: customers, logs: logs}
}

func (s *CustomerService) Create(customer Customer) (Customer, error) {
	if err := validateCustomer(customer); err != nil {
		return Customer{}, err
	}
	created, err := s.customers.Create(customer)
	if err != nil {
		return Customer{}, err
	}
	s.logs.Append(OperationLog{Action: "customer.create", Target: created.ID, Detail: created.Phone})
	return created, nil
}

func (s *CustomerService) Update(id string, customer Customer) (Customer, error) {
	if _, err := s.customers.Get(id); err != nil {
		return Customer{}, err
	}
	customer.ID = id
	if err := validateCustomer(customer); err != nil {
		return Customer{}, err
	}
	if err := s.customers.Put(customer); err != nil {
		return Customer{}, err
	}
	s.logs.Append(OperationLog{Action: "customer.update", Target: id, Detail: customer.Phone})
	return customer, nil
}

func (s *CustomerService) List() []Customer {
	return s.customers.List()
}

func (s *CustomerService) Logs() []OperationLog {
	return s.logs.List()
}

func (s *CustomerService) PreviewImport(records []CustomerImportRecord) ImportPreview {
	preview := ImportPreview{Records: append([]CustomerImportRecord(nil), records...)}

	var phoneCounts map[string]int
	for _, record := range records {
		phoneCounts[record.Phone]++
	}

	existing := make(map[string]bool)
	for _, customer := range s.customers.List() {
		existing[customer.Phone] = true
	}
	duplicates := make(map[string]bool)
	for phone, count := range phoneCounts {
		if count > 1 {
			duplicates[phone] = true
		}
		if existing[phone] {
			preview.ExistingPhones = append(preview.ExistingPhones, phone)
		}
	}
	for phone := range duplicates {
		preview.DuplicatePhones = append(preview.DuplicatePhones, phone)
	}
	sort.Strings(preview.DuplicatePhones)
	sort.Strings(preview.ExistingPhones)
	s.logs.Append(OperationLog{Action: "customer.import.preview", Target: "workbook", Detail: fmt.Sprintf("%d records", len(records))})
	return preview
}

func (s *CustomerService) CommitImport(records []CustomerImportRecord) ([]Customer, error) {
	preview := s.PreviewImport(records)
	if len(preview.DuplicatePhones) > 0 || len(preview.ExistingPhones) > 0 {
		return nil, ErrDuplicatePhone
	}
	created := make([]Customer, 0, len(records))
	for _, record := range records {
		customer, err := s.Create(record.customer(""))
		if err != nil {
			return nil, err
		}
		created = append(created, customer)
	}
	s.logs.Append(OperationLog{Action: "customer.import", Target: "batch", Detail: fmt.Sprintf("%d records", len(created))})
	return created, nil
}

func validateCustomer(customer Customer) error {
	if strings.TrimSpace(customer.Name) == "" || strings.TrimSpace(customer.Phone) == "" || strings.TrimSpace(customer.PatientName) == "" || strings.TrimSpace(customer.ServiceCity) == "" {
		return ErrInvalidCustomer
	}
	return nil
}
