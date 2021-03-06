package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	ports "github.com/yittg/ving/addons/port/config"
	statistic "github.com/yittg/ving/statistic/config"
	ui "github.com/yittg/ving/ui/config"
)

var searchDir = []string{".", os.Getenv("HOME")}

// Config custom
type Config struct {
	AddOns    AddOnConfig `toml:"add-ons"`
	UI        ui.UIConfig
	Statistic statistic.Config
}

// AddOnConfig add on configs
type AddOnConfig struct {
	Ports ports.PortsConfig
}

var customConfig *Config

// GetConfig get custom config
func GetConfig() *Config {
	return customConfig
}

func validateAddOnConfig(ac *AddOnConfig) error {
	return ac.Ports.Validate()
}

func validate(c *Config) error {
	if err := c.UI.Validate(); err != nil {
		return err
	}
	if err := c.Statistic.Validate(); err != nil {
		return err
	}
	return validateAddOnConfig(&c.AddOns)
}

func init() {
	customConfig = &Config{
		AddOns: AddOnConfig{
			Ports: ports.Default(),
		},
		UI:        ui.Default(),
		Statistic: statistic.Default(),
	}
	for _, rcDir := range searchDir {
		rcFile := rcDir + "/.ving.toml"
		if _, err := os.Stat(rcFile); os.IsNotExist(err) {
			continue
		}
		if _, err := toml.DecodeFile(rcFile, customConfig); err != nil {
			panic(err)
		}
		if err := validate(customConfig); err != nil {
			fmt.Printf("Invalid custom configuration file: %s\n%s\n", rcFile, err)
			os.Exit(1)
		}
		return
	}
}
