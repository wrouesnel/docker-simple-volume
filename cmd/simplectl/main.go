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
	"sort"
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

type checkVolumeQueryCmd struct {
	targetDevice string
	inputQueryString volumequery.VolumeQuery
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

	listDevicePartitions := app.Command("list-partitions", "list all partitions from a device")
	listDevicePartitionsCmd := listDevicePartitionsCmd{}
	listDevicePartitions.Arg("disk path", "disk or device to reverse engineer selection rules for").StringVar(&listDevicePartitionsCmd.targetDevice)

	listRawCandidates := app.Command("list-raw-candidates", "list devices which form the initial pool")
	listInitializedCandidates := app.Command("list-initialized-candidates", "list initialized simple devices")
	listUninitializedCandidates := app.Command("list-uninitialized-candidates", "list uninitialized devices simple would initialize on request")
	listRejectedCandidates := app.Command("list-rejected-candidates", "list candidate devices which were rejected for one reason or another")

	forceInitDisk := app.Command("initialize-disk", "manually write an initialization value to a given block device")
	forceInitCmdData := forceInitCmd{}
	forceInitDisk.Flag("force", "don't prompt for confirmation").BoolVar(&forceInitCmdData.force)
	forceInitDisk.Flag("hostname", "override hostname for disk").Default(hostname()).StringVar(&forceInitCmdData.hostname)
	forceInitDisk.Flag("machine-id", "override machine-id for disk").Default(machineid()).StringVar(&forceInitCmdData.machineid)
	forceInitDisk.Arg("block device","block device to partition and initialize").StringVar(&forceInitCmdData.targetDevice)
	volumequery.VolumeQueryVar(forceInitDisk.Arg("initializing query string", "query string used to initialize the device"), &forceInitCmdData.inputQueryString)

	checkVolumeQuery := app.Command("query-device", "run a given query string against a given device")
	checkVolumeQueryCmdData := checkVolumeQueryCmd{}
	checkVolumeQuery.Arg("block device","block device to partition and initialize").StringVar(&checkVolumeQueryCmdData.targetDevice)
	volumequery.VolumeQueryVar(checkVolumeQuery.Arg("query string", "query string used to initialize the device"), &checkVolumeQueryCmdData.inputQueryString)

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
		devices, err := volumequery.GetDevicePaths(jsonRules)
		if err != nil {
			log.Fatalln(err)
		}
		for _, d := range devices {
			fmt.Println(d)
		}

	case dumpDeviceRules.FullCommand():
		rule, err := volumequery.GetFullSelectionRuleForDevice(dumpDeviceRulesCmd.targetDevice)
		if err != nil {
			log.Fatalln("Failed to query device:", err)
		}
		rules := []*volumequery.DeviceSelectionRule{rule}
		b, err := json.MarshalIndent(rules,""," ")
		if err != nil {
			log.Fatalln("JSON marshalling failed:", err)
		}
		os.Stdout.Write(b)
		os.Stdout.Write([]byte{'\n'})

	case listRawCandidates.FullCommand():
		devices, err := volumequery.GetDevicePaths([]volumequery.DeviceSelectionRule{cmdlineSelectionRule})
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
		devPaths := make([]string, 0, len(devices))
		for d, _ := range devices {
			devPaths = append(devPaths, d)
		}
		sort.Strings(devPaths)
		for _, d := range devPaths {
			fmt.Println(d)
		}

	case listUninitializedCandidates.FullCommand():
		_, uninitialized, _, err := volumequery.GetCandidateDisks([]volumequery.DeviceSelectionRule{cmdlineSelectionRule})
		if err != nil {
			log.Fatalln("Failed while querying candidates:", err)
		}
		fmt.Fprintln(os.Stderr, "Listing uninitialized candidate devices for simple")
		for _, d := range uninitialized {
			fmt.Println(d)
		}

	case listInitializedCandidates.FullCommand():
		initialized, _, _, err := volumequery.GetCandidateDisks([]volumequery.DeviceSelectionRule{cmdlineSelectionRule})
		if err != nil {
			log.Fatalln("Failed while querying candidates:", err)
		}
		fmt.Fprintln(os.Stderr, "Listing initialized candidate devices for simple")
		for _, d := range initialized {
			fmt.Println(d)
		}

	case listRejectedCandidates.FullCommand():
		_, _, rejected, err := volumequery.GetCandidateDisks([]volumequery.DeviceSelectionRule{cmdlineSelectionRule})
		if err != nil {
			log.Fatalln("Failed while querying candidates:", err)
		}
		fmt.Fprintln(os.Stderr, "Listing rejected candidate devices for simple")
		for _, d := range rejected {
			fmt.Println(d)
		}

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

	case checkVolumeQuery.FullCommand():
		fmt.Fprintln(os.Stderr, "Checking query against device:", checkVolumeQueryCmdData.targetDevice)
		labelPath, dataPath, err := volumequery.GetDiskLabelAndVolumePath(checkVolumeQueryCmdData.targetDevice)
		if err != nil {
			log.Fatalln("Not an initialized or locateable device:", err)
		}
		if matches, err := volumequery.VolumeQueryMatch(&checkVolumeQueryCmdData.inputQueryString, labelPath, dataPath); err != nil {
			log.Fatalln("Error while trying to run matcher:", err)
		} else if matches {
			fmt.Fprintln(os.Stdout, "Match")
		} else {
			fmt.Fprintln(os.Stdout, "No Match")
		}
	}

}