package trigger

import (
	"github.com/sirupsen/logrus"
	"m0rg.dev/x10/runner"
)

type CommandTrigger struct{}
type CommandTriggerData struct {
	Script string
}

func init() {
	RegisterTrigger(CommandTrigger{}, "command")
}

func (CommandTrigger) RunInstall(logger *logrus.Entry, raw_data interface{}) error {
	data := CommandTriggerData{}
	raw_data_map := raw_data.(map[interface{}]interface{})
	if raw_data_map["script"] != nil {
		data.Script = raw_data_map["script"].(string)
	}

	return runner.RunTargetScript(logger, data.Script, []string{})
}
