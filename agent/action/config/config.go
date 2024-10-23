package config

import (
	cfg2 "xprobe_agent/config"
	"xprobe_agent/log"
)

type ConfigAction struct {
}

func (a *ConfigAction) Execute(name string, data interface{}) error {
	cfg2.SetOtherConfig(name, data.(string))
	log.Infof("set config: %s = %s", name, data.(string))
	// panic("test")
	return nil
}
