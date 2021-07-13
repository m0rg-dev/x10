package trigger

import (
	"github.com/sirupsen/logrus"
	"m0rg.dev/x10/spec"
	"m0rg.dev/x10/x10_log"
)

type Trigger interface {
	RunInstall(logger *logrus.Entry, data interface{}) error
}

var triggers = map[string]Trigger{}

func RegisterTrigger(t Trigger, name string) {
	triggers[name] = t
}

func RunTriggers(pkg spec.SpecLayer) error {
	logger := x10_log.Get("trigger").WithField("pkg", pkg.GetFQN())
	for name, t := range triggers {
		data, ok := pkg.TriggerData[name]
		if ok {
			err := t.RunInstall(logger.WithField("trigger", name), data)
			// TODO think about error handling in triggers, in general
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func iarrayconv(in []interface{}) []string {
	ret := make([]string, len(in))
	for i, v := range in {
		ret[i] = v.(string)
	}
	return ret
}
