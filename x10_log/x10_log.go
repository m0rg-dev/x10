package x10_log

import (
	"os"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
)

func Get(what string) *logrus.Entry {
	log := logrus.New()
	log.SetOutput(os.Stderr)
	if _, ok := os.LookupEnv("X10_DEBUG"); ok {
		log.SetLevel(logrus.DebugLevel)
	}
	log.SetFormatter(&nested.Formatter{
		HideKeys:    true,
		FieldsOrder: []string{"pkg", "what"},
	})
	return log.WithField("what", what)
}
