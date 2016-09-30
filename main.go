/*
	Docker simple volume driver
*/

package main

import (
	"flag"
	"fmt"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/wrouesnel/docker-simple-disk/fsutil"
	"github.com/wrouesnel/go.log"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"sync"
	"github.com/go-errors/errors"
)

const (
	PluginName string = "simple"
)

var Version string = "dev"

type SimpleVolumeDriver struct {
	volumeRoot string
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

func NewSimpleVolumeDriver(volumeRoot string) *SimpleVolumeDriver {
	return &SimpleVolumeDriver{
		volumeRoot: volumeRoot,
	}
}

func main() {
	dockerPluginPath := kingpin.Flag("docker-plugins", "Listen path for the plugin.").Default(fmt.Sprintf("unix:///run/docker/plugins/%s.sock", PluginName)).URL()
	volumeRoot := kingpin.Flag("volume-root", "Path where mounted volumes should be created").Default("/tmp/docker-simple").String()
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
			log.Panicln("socket-root does not exist.")
		}
	} else if !fsutil.PathIsDir(*volumeRoot) {
		log.Panicln("volume-root exists but is not a directory.")
	}

	log.Infoln("Volume mount root:", *volumeRoot)
	log.Infoln("Docker Plugin Path:", *dockerPluginPath)

	driver := NewSimpleVolumeDriver(*volumeRoot)
	handler := volume.NewHandler(driver)

	if err := handler.ServeUnix("root", PluginName) ; err != nil {
		log.Errorln(err)
	}
}
