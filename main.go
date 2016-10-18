/*
	Docker simple volume driver
*/

package main

import (
	"flag"
	"fmt"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/go-errors/errors"
	"github.com/wrouesnel/docker-simple-disk/fsutil"
	"github.com/wrouesnel/go.log"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"sync"
)

const (
	PluginName string = "simple"
)

var Version string = "dev"

type SimpleVolumeDriver struct {
	// Root directory to mount volumes at
	volumeRoot string
	// Device selection rules
	deviceSelectionRules []DeviceSelectionRule
	// Mutex to serialize volume operations
	mtx sync.RWMutex
}

type SimpleVolume struct {
	// Canonical docker name used to create us
	name string
	// Compact UUID type format of the volume
	typeid string
}

// On create, check we can service the request, setup the staging volume
// for the request.
func (this *SimpleVolumeDriver) Create(req volume.Request) volume.Response {
	log.Debugln("Create:", req)
	return volume.Response{
		Err: errors.Errorf("Not implemented: %v", req).Error(),
	}
}

func (this *SimpleVolumeDriver) List(req volume.Request) volume.Response {
	log.Debugln("List:", req)
	return volume.Response{
		Err: errors.Errorf("Not implemented: %v", req).Error(),
	}
}

func (this *SimpleVolumeDriver) Get(req volume.Request) volume.Response {
	log.Debugln("Get:", req)
	return volume.Response{
		Err: errors.Errorf("Not implemented: %v", req).Error(),
	}
}

func (this *SimpleVolumeDriver) Remove(req volume.Request) volume.Response {
	log.Debugln("Remove:", req)
	return volume.Response{
		Err: errors.Errorf("Not implemented: %v", req).Error(),
	}
}

func (this *SimpleVolumeDriver) Path(req volume.Request) volume.Response {
	log.Debugln("Path:", req)
	return volume.Response{
		Err: errors.Errorf("Not implemented: %v", req).Error(),
	}
}

func (this *SimpleVolumeDriver) Mount(req volume.MountRequest) volume.Response {
	log.Debugln("Mount:", req)
	return volume.Response{
		Err: errors.Errorf("Not implemented: %v", req).Error(),
	}
}

func (this *SimpleVolumeDriver) Unmount(req volume.UnmountRequest) volume.Response {
	log.Debugln("Unmount:", req)
	return volume.Response{
		Err: errors.Errorf("Not implemented: %v", req).Error(),
	}
}

func (this *SimpleVolumeDriver) Capabilities(req volume.Request) volume.Response {
	log.Debugln("Capabilities:", req)
	return volume.Response{
		Err: errors.Errorf("Not implemented: %v", req).Error(),
	}
}

func NewSimpleVolumeDriver(volumeRoot string, deviceSelectionRules []DeviceSelectionRule) *SimpleVolumeDriver {
	return &SimpleVolumeDriver{
		volumeRoot:           volumeRoot,
		deviceSelectionRules: deviceSelectionRules,
	}
}

func main() {
	dockerPluginPath := kingpin.Flag("docker-plugins", "Listen path for the plugin.").Default(fmt.Sprintf("unix:///run/docker/plugins/%s.sock", PluginName)).URL()
	volumeRoot := kingpin.Flag("volume-root", "Path where mounted volumes should be created").Default("/tmp/docker-simple").String()

	// Various udev matching options and some sane defaults for most users
	cmdlineSelectionRule := DeviceSelectionRule{}
	kingpin.Flag("device-match-subsystem", "udev subsystem match for finding elegible devices").Default("block").StringsVar(&cmdlineSelectionRule.Subsystems)
	kingpin.Flag("device-match-name", "udev name to match for finding elegible devices").Default("sd*").StringsVar(&cmdlineSelectionRule.Name)
	kingpin.Flag("device-match-tag", "udev tag to match for finding elegible devices").StringsVar(&cmdlineSelectionRule.Tag)

	kingpin.Flag("device-match-attr", "udev sys attribute to match for finding elegible devices").StringMapVar(&cmdlineSelectionRule.Attrs)
	kingpin.Flag("device-match-properties", "udev property to match for finding elegible devices (i.e. environment variables)").Default("DEVTYPE=disk").StringMapVar(&cmdlineSelectionRule.Properties)

	loglevel := kingpin.Flag("log-level", "Logging Level").Default("info").String()
	logformat := kingpin.Flag("log-format", "If set use a syslog logger or JSON logging. Example: logger:syslog?appname=bob&local=7 or logger:stdout?json=true. Defaults to stderr.").Default("stderr").String()
	kingpin.Parse()

	// Check for the programs we need to actually work
	fsutil.MustLookupPaths(
		"sgdisk",
		"mkfs",
	)

	flag.Set("log.level", *loglevel)
	flag.Set("log.format", *logformat)

	if !fsutil.PathExists(*volumeRoot) {
		err := os.MkdirAll(*volumeRoot, os.FileMode(0777))
		if err != nil {
			log.Panicln("volume-root does not exist and could not be created.")
		}
	} else if !fsutil.PathIsDir(*volumeRoot) {
		log.Panicln("volume-root exists but is not a directory.")
	}

	log.Infoln("Volume mount root:", *volumeRoot)
	log.Infoln("Docker Plugin Path:", *dockerPluginPath)

	driver := NewSimpleVolumeDriver(*volumeRoot,
		[]DeviceSelectionRule{cmdlineSelectionRule})
	handler := volume.NewHandler(driver)

	if err := handler.ServeUnix("root", PluginName); err != nil {
		log.Errorln(err)
	}
}
