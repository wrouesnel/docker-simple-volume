// simplectl is a debugging/administration utility. It allows access to most of
// the backend functionality of simple from the command line.

package main

import (
	"os"
	"fmt"

	"gopkg.in/alecthomas/kingpin.v2"
	"github.com/wrouesnel/go.log"

	"github.com/wrouesnel/docker-simple-disk/volumequery"
	//"github.com/wrouesnel/docker-simple-disk/volumelabel"
	"github.com/wrouesnel/docker-simple-disk/config"
	"github.com/coreos/go-systemd/util"
	"github.com/Songmu/prompter"
	"github.com/wrouesnel/docker-simple-disk/volumesetup"
	"encoding/json"
	"io/ioutil"
)

type dumpDeviceRulesCmd struct {
	targetDevice string
}

type listDevicePartitionsCmd struct {
	targetDevice string
}

type forceInitCmd struct {
	targetDevice string
	inputQueryString volumequery.VolumeQuery
	force bool
	hostname string
	machineid string
}

// Get the hostname
func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return ""
	}
	return h
}

// Get the machine ID
func machineid() string {
	m, err := util.GetMachineID()
	if err != nil {
		return ""
	}
	return m
}

func main() {
	app := kingpin.New("simplectl", "Test utility which runs simple volume queries against udev")

	cmdlineSelectionRule := volumequery.NewDeviceSelectionRule()
	config.DefaultAppFlags(app, &cmdlineSelectionRule)

	rawQueryFromStdin := app.Command("raw-udev-query", "Read a JSON selection rules from stdin and run a udev query")

	dumpDeviceRules := app.Command("dump-device-rules", "JSON print the parameter of a device as selection rules")
	dumpDeviceRulesCmd := dumpDeviceRulesCmd{}
	dumpDeviceRules.Arg("disk path", "disk or device to reverse engineer selection rules for").StringVar(&dumpDeviceRulesCmd.targetDevice)

	listRawCandidates := app.Command("list-raw-candidates", "list all possible candidate devices")

	//listCandidates := app.Command("list-candidates", "list currently selectable candidate devices (not mounted)")

	listDevicePartitions := app.Command("list-partitions", "list all partitions from a device")
	listDevicePartitionsCmd := listDevicePartitionsCmd{}
	listDevicePartitions.Arg("disk path", "disk or device to reverse engineer selection rules for").StringVar(&listDevicePartitionsCmd.targetDevice)

	forceInitDisk := app.Command("initialize-disk", "manually write an initialization value to a given block device")
	forceInitCmdData := forceInitCmd{}
	forceInitDisk.Flag("force", "don't prompt for confirmation").BoolVar(&forceInitCmdData.force)
	forceInitDisk.Flag("hostname", "override hostname for disk").Default(hostname()).StringVar(&forceInitCmdData.hostname)
	forceInitDisk.Flag("machine-id", "override machine-id for disk").Default(machineid()).StringVar(&forceInitCmdData.machineid)
	forceInitDisk.Arg("block device","block device to partition and initialize").StringVar(&forceInitCmdData.targetDevice)
	volumequery.VolumeQueryVar(forceInitDisk.Arg("initializing query string", "query string used to initialize the device"), &forceInitCmdData.inputQueryString)

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case rawQueryFromStdin.FullCommand():
		// Run GetCandidateDisk but read from a JSON query.
		jsonBytes, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalln("Error reading JSON from stdin.")
		}
		jsonRules := []volumequery.DeviceSelectionRule{}
		if err := json.Unmarshal(jsonBytes, &jsonRules); err != nil {
			log.Fatalln("Error unmarshalling query from JSON:", err)
		}
		devices, err := volumequery.GetCandidateDisks(jsonRules)
		if err != nil {
			log.Fatalln(err)
		}
		for _, d := range devices {
			fmt.Println(d)
		}

	case dumpDeviceRules.FullCommand():
		rules, err := volumequery.GetFullSelectionRulesForDevice(dumpDeviceRulesCmd.targetDevice)
		if err != nil {
			log.Fatalln("Failed to query device:", err)
		}
		b, err := json.MarshalIndent(rules,""," ")
		if err != nil {
			log.Fatalln("JSON marshalling failed:", err)
		}
		os.Stdout.Write(b)
		os.Stdout.Write([]byte{'\n'})

	case listRawCandidates.FullCommand():
		devices, err := volumequery.GetCandidateDisks([]volumequery.DeviceSelectionRule{cmdlineSelectionRule})
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Fprintln(os.Stderr, "Listing ALL possible candidate devices (not filtered for usage)")
		for _, d := range devices {
			fmt.Println(d)
		}

	case listDevicePartitions.FullCommand():
		devices, err := volumequery.GetPartitionDevicesFromDiskPath(listDevicePartitionsCmd.targetDevice)
		if err != nil {
			log.Fatalln("Failed to query device:", err)
		}
		fmt.Fprintln(os.Stderr, "Listing partitions of device")
		for _, d := range devices {
			fmt.Println(d)
		}

	//case listCandidates.FullCommand():
	//	devices, err := volumequery.GetAvailableCandidateDisks([]volumequery.DeviceSelectionRule{cmdlineSelectionRule})
	//	if err != nil {
	//		log.Fatalln(err)
	//	}
	//	fmt.Fprintln(os.Stderr, "Listing available candidate devices")

	case forceInitDisk.FullCommand():
		if !forceInitCmdData.force == false {
			if proceed := prompter.YesNo("Force initializing the given device. Are you sure?", false); !proceed {
				log.Fatalln("Cancelled by user.")
			}
		}
		log.Infoln("Forcibly initializing device:", forceInitCmdData.targetDevice)
		err := volumesetup.InitializeBlockDevice(
			forceInitCmdData.targetDevice,
			forceInitCmdData.inputQueryString,
			forceInitCmdData.hostname,
			forceInitCmdData.machineid,
		)
		if err != nil {
			log.Fatalln("Failed while setting up device:", err)
		}
	}

}