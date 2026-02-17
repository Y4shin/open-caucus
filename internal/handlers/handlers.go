package handlers

import (
	"fmt"
	"strings"

	"github.com/Y4shin/conference-tool/internal/broker"
	"github.com/Y4shin/conference-tool/internal/repository"
	"github.com/Y4shin/conference-tool/internal/session"
)

type HandlerBuilder struct {
	broker     broker.Broker
	repository repository.Repository
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

func (b *HandlerBuilder) Build() *Handler {
	missing := []string{}
	if b.broker == nil {
		missing = append(missing, "broker")
	}
	if b.repository == nil {
		missing = append(missing, "repository")
	}
	if len(missing) > 0 {
		missingStr := strings.Join(missing, ", ")
		panic(fmt.Sprintf("cannot build Handler with missing %s", missingStr))
	}
	return &Handler{
		Broker:     b.broker,
		Repository: b.repository,
	}
}

type Handler struct {
	Broker              broker.Broker
	Repository          repository.Repository
	SessionManager      *session.Manager
	AdminSessionManager *session.AdminSessionManager
	AdminKey            string
}

func NewHandler(b broker.Broker) *Handler {
	return &Handler{Broker: b}
}
