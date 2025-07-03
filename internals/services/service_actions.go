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
	ctx.svc.Metadata.FailureCount++

	if ctx.svc.Metadata.FailureCount > 3 {
		ctx.svc.Logger.Warn("Service %s has failed for 3 consecutive health checks", ctx.svc.Name)
		return internals.NOOP
	}

	return internals.NOOP
}
