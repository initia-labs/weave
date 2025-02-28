package context

import (
	"context"
	"fmt"

	"github.com/initia-labs/weave/service"
)

func SetService(ctx context.Context, srv service.Service) context.Context {
	return context.WithValue(ctx, ServiceKey, srv)
}

func GetService(ctx context.Context) (service.Service, error) {
	srv, ok := ctx.Value(ServiceKey).(service.Service)
	if !ok {
		return nil, fmt.Errorf("service not found in context")
	}
	return srv, nil
}
