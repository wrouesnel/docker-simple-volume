// helper utilty to test query strings against current system.

package main

import (
	"os"

	"gopkg.in/alecthomas/kingpin.v2"
	"github.com/wrouesnel/go.log"

	"github.com/wrouesnel/docker-simple-disk/volumequery"
	//"github.com/wrouesnel/docker-simple-disk/volumelabel"
	"github.com/wrouesnel/docker-simple-disk/config"
	"fmt"
)

type forceInitCmd struct {
	targetDevice string
}

func main() {
	app := kingpin.New("simple-test-query", "Test utility which runs simple volume queries against udev")

	cmdlineSelectionRule := volumequery.NewDeviceSelectionRule()
	config.DefaultAppFlags(app, &cmdlineSelectionRule)

	listRawCandidates := app.Command("list-raw-candidates", "list all possible candidate devices")

	forceInitCmdData := forceInitCmd{}
	forceInitDisk := app.Command("initialize-disk", "manually write an initialization value to a given block device")
	forceInitDisk.Arg("block device","block device to partition and initialize").StringVar(&forceInitCmdData.targetDevice)

	//queryString := app.Arg("query string", "docker volume string to run a query for")

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case listRawCandidates.FullCommand():
		devices, err := volumequery.GetDevicesByDevNode([]volumequery.DeviceSelectionRule{cmdlineSelectionRule})
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Fprintln(os.Stderr, "Listing ALL possible candidate devices (not filtered for usage)")
		for _, d := range devices {
			fmt.Println(d)
		}
	case forceInitDisk.FullCommand():

	}

	//query := &volumequery.VolumeQuery{}
	//
	//err := volumelabel.UnmarshalVolumeLabel(*queryString, query)
	//if err != nil {
	//	log.Fatalln("Failed unmarshalling query in volume label format:", err)
	//}

	//
}