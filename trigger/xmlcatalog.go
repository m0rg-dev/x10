package trigger

import (
	"github.com/sirupsen/logrus"
	"m0rg.dev/x10/runner"
)

type XmlCatalogTrigger struct{}
type XmlCatalogTriggerData struct {
	SgmlEntries []string
	XmlEntries  []string
}

func init() {
	RegisterTrigger(XmlCatalogTrigger{}, "xmlcatalog")
}

func (XmlCatalogTrigger) RunInstall(logger *logrus.Entry, raw_data interface{}) error {
	data := XmlCatalogTriggerData{}
	raw_data_map := raw_data.(map[interface{}]interface{})
	if raw_data_map["sgmlentries"] != nil {
		data.SgmlEntries = iarrayconv(raw_data_map["sgmlentries"].([]interface{}))
	}
	if raw_data_map["xmlentries"] != nil {
		data.XmlEntries = iarrayconv(raw_data_map["xmlentries"].([]interface{}))
	}

	if data.SgmlEntries != nil {
		logger.Info("Registering SGML catalog entries...")
		for _, entry := range data.SgmlEntries {
			logger.Info(entry)
			err := runner.RunTargetScript(logger, "/usr/bin/xmlcatmgr -sc /usr/share/sgml/catalog add "+entry, []string{})
			if err != nil {
				return err
			}
		}
	}

	if data.XmlEntries != nil {
		logger.Info("Registering XML catalog entries...")
		for _, entry := range data.XmlEntries {
			logger.Info(entry)
			err := runner.RunTargetScript(logger, "/usr/bin/xmlcatmgr -c /usr/share/xml/catalog add "+entry, []string{})
			if err != nil {
				return err
			}
		}
	}

	return nil
}
