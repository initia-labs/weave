package context

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/initia-labs/weave/common"
)

func SetInitiaHome(ctx context.Context, initiaHome string) context.Context {
	return context.WithValue(ctx, InitiaHomeKey, initiaHome)
}

func GetInitiaHome(ctx context.Context) (string, error) {
	if value, ok := ctx.Value(InitiaHomeKey).(string); ok {
		return value, nil
	}
	return "", fmt.Errorf("cannot cast the InitiaHomeKey value into type string")
}

func GetInitiaConfigDirectory(ctx context.Context) (string, error) {
	initiaHome, err := GetInitiaHome(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(initiaHome, common.InitiaConfigDirectory), nil
}

func GetInitiaDataDirectory(ctx context.Context) (string, error) {
	initiaHome, err := GetInitiaHome(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(initiaHome, common.InitiaDataDirectory), nil
}

func SetMinitiaHome(ctx context.Context, minitiaHome string) context.Context {
	return context.WithValue(ctx, MinitiaHomeKey, minitiaHome)
}

func GetMinitiaHome(ctx context.Context) (string, error) {
	if value, ok := ctx.Value(MinitiaHomeKey).(string); ok {
		return value, nil
	}
	return "", fmt.Errorf("cannot cast the MinitiaHomeKey value into type string")
}

func GetMinitiaArtifactsConfigJson(ctx context.Context) (string, error) {
	minitiaHome, err := GetMinitiaHome(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(minitiaHome, common.MinitiaArtifactsConfigJson), nil
}

func GetMinitiaArtifactsJson(ctx context.Context) (string, error) {
	minitiaHome, err := GetMinitiaHome(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(minitiaHome, common.MinitiaArtifactsJson), nil
}

func SetOPInitHome(ctx context.Context, opInitHome string) context.Context {
	return context.WithValue(ctx, OPInitHomeKey, opInitHome)
}

func GetOPInitHome(ctx context.Context) (string, error) {
	if value, ok := ctx.Value(OPInitHomeKey).(string); ok {
		return value, nil
	}
	return "", fmt.Errorf("cannot cast the OPInitHomeKey value into type string")
}

func GetOPInitKeyFileJson(ctx context.Context) (string, error) {
	opInitHome, err := GetOPInitHome(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(opInitHome, common.OPinitKeyFileJson), nil
}
