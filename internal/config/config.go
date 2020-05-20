package config

import (
	"context"

	"github.com/mvndaai/ctxerr"
	"github.com/spf13/viper"
)

// config value
const (
	// Personal Info
	Email       = "email"
	FirstName   = "first-name"
	LastName    = "last-name"
	PhoneNumber = "phone-number"
	ZipCode     = "zip-code"
)

// InitConfig setups and reads in the config
func InitConfig(ctx context.Context) error {
	viper.SetDefault(Email, "")
	viper.SetDefault(FirstName, "")
	viper.SetDefault(LastName, "")
	viper.SetDefault(PhoneNumber, "")
	viper.SetDefault(ZipCode, "")

	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	// viper.WriteConfigAs("example-config.json")

	err := viper.ReadInConfig()
	if err != nil {
		ctxerr.QuickWrap(ctx, err)
	}
	return nil
}
