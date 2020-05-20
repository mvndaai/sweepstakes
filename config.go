package main

import (
	"context"

	"github.com/mvndaai/ctxerr"
	"github.com/spf13/viper"
)

func initConfig(ctx context.Context) error {
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()
	if err != nil {
		ctxerr.QuickWrap(ctx, err)
	}
	return nil
}

// TODO use go generate to create the example config
