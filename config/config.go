package config

import (

)
import (
	"gopkg.in/alecthomas/kingpin.v2"
	"github.com/wrouesnel/docker-simple-disk/volumequery"
	"flag"
)

// DefaultAppFlags sets up the mapping of default app configuration which the
// underlying subsystems always need.
func DefaultAppFlags(app *kingpin.Application, cmdlineSelectionRule *volumequery.DeviceSelectionRule) {
	// Default device match subsystem
	app.Flag("device-match-subsystem", "udev subsystem match for finding elegible devices").Default("block").StringsVar(&cmdlineSelectionRule.Subsystems)
	app.Flag("device-match-name", "udev name to match for finding elegible devices").Default("sd*").StringsVar(&cmdlineSelectionRule.Name)
	app.Flag("device-match-tag", "udev tag to match for finding elegible devices").StringsVar(&cmdlineSelectionRule.Tag)

	app.Flag("device-match-attr", "udev sys attribute to match for finding elegible devices").StringMapVar(&cmdlineSelectionRule.Attrs)
	app.Flag("device-match-properties", "udev property to match for finding elegible devices (i.e. environment variables)").Default("DEVTYPE=disk").StringMapVar(&cmdlineSelectionRule.Properties)

	// Handle logging globally
	loglevel := app.Flag("log-level", "Logging Level").Default("info").String()
	logformat := app.Flag("log-format", "If set use a syslog logger or JSON logging. Example: logger:syslog?appname=bob&local=7 or logger:stdout?json=true. Defaults to stderr.").Default("stderr").String()

	app.Action(func(*kingpin.ParseContext) error {
		flag.Set("log.level", *loglevel)
		flag.Set("log.format", *logformat)
		return nil
	})
}