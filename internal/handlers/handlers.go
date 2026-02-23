package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Y4shin/conference-tool/internal/broker"
	"github.com/Y4shin/conference-tool/internal/pagination"
	"github.com/Y4shin/conference-tool/internal/repository"
	"github.com/Y4shin/conference-tool/internal/session"
	"github.com/Y4shin/conference-tool/internal/storage"
)

type HandlerBuilder struct {
	broker     broker.Broker
	repository repository.Repository
	storage    storage.Service
}

func (b *HandlerBuilder) WithBroker(broker broker.Broker) *HandlerBuilder {
	if b.broker != nil {
		panic("trying to set multiple brokers")
	}
	if broker == nil {
		panic("cannot set nil broker")
	}
	b.broker = broker
	return b
}

func (b *HandlerBuilder) WithRepository(repository repository.Repository) *HandlerBuilder {
	if b.repository != nil {
		panic("trying to set multiple repositorys")
	}
	if repository == nil {
		panic("cannot set nil repository")
	}
	b.repository = repository
	return b
}

func (b *HandlerBuilder) WithStorage(svc storage.Service) *HandlerBuilder {
	if b.storage != nil {
		panic("trying to set multiple storage services")
	}
	if svc == nil {
		panic("cannot set nil storage service")
	}
	b.storage = svc
	return b
}

func (b *HandlerBuilder) Build() *Handler {
	missing := []string{}
	if b.broker == nil {
		missing = append(missing, "broker")
	}
	if b.repository == nil {
		missing = append(missing, "repository")
	}
	if b.storage == nil {
		missing = append(missing, "storage")
	}
	if len(missing) > 0 {
		missingStr := strings.Join(missing, ", ")
		panic(fmt.Sprintf("cannot build Handler with missing %s", missingStr))
	}
	return &Handler{
		Broker:     b.broker,
		Repository: b.repository,
		Storage:    b.storage,
	}
}

type Handler struct {
	Broker         broker.Broker
	Repository     repository.Repository
	Storage        storage.Service
	SessionManager *session.Manager
}

func NewHandler(b broker.Broker) *Handler {
	return &Handler{Broker: b}
}

// parsePaginationParams reads ?page and ?page_size from the request, applying sensible defaults.
func parsePaginationParams(r *http.Request) (page, pageSize int) {
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ = strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 {
		pageSize = pagination.DefaultPageSize
	}
	return page, pageSize
}
