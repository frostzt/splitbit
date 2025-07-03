package services

import "github.com/frostzt/splitbit/internals"

type CommonActionCtx struct {
	svc *Service
}

type ServiceAliveAction struct{}

func (a *ServiceAliveAction) Execute(eventCtx internals.EventContext) internals.EventType {
	ctx := eventCtx.(*CommonActionCtx)

	ctx.svc.Logger.Debug("Received service alive event for %s", ctx.svc.Name)

	// Reset the failure count
	ctx.svc.Metadata.FailureCount = 0
	return internals.NOOP
}

type ServiceDownAction struct{}

func (a *ServiceDownAction) Execute(eventCtx internals.EventContext) internals.EventType {
	ctx := eventCtx.(*CommonActionCtx)

	ctx.svc.Logger.Debug("Received service down event for %s", ctx.svc.Name)

	// Reset the failure count
	ctx.svc.Metadata.FailureCount = 0
	return internals.NOOP
}
