package main

import (
	"github.com/jochenvg/go-udev"
	"github.com/jkeiser/iter"
)

type NamingType int

const (
	NamingNumeric int = 0
	NamingUUID int = iota
)

// Specifies a volume query (this is a mash-up of query and create parameters
// in reality.
type VolumeQuery struct {
	Label string

	Hostname string
	MachineId string

	Initialized bool
	Exclusive bool
	DynamicMounts bool
	PersistNumbering bool

	Basename string
	NamingStyle NamingType

	MinimumSizeBytes uint64
	MaximumSizeBytes uint64

	MinDisks	int
	MaxDisks	int

	Filesystem string

	Metadata map[string]string
}

// Struct representing labelled data (output as JSON)
type VolumeLabel struct {
	// Hostname this disk was last initialized on
	Hostname string	`json:"hostname"`
	// Machine ID this disk was last initialized on, if available
	MachineId string	`json:"machine_id"`
	// Label of this disk (should match partition label)
	Label string `json:"label"`
	// Last numbering assignment this disk had for the current label
	Numbering string	`json:"numbering"`
	// Extra metadata
	Metadata map[string]string `json:"metadata"`
}

// Struct representing a disk which is able to be used by the plugin
type Disk struct {
	Label VolumeLabel
	PartitionPath string
}

// Get a list of disks on the current host which can be considered for use by
// the plugin, and return their device paths.
func GetDisks() ([]string, error) {
	udevCtx := udev.Udev{}

	deviceEnumerator := udevCtx.NewEnumerate()

	if err := deviceEnumerator.AddMatchIsInitialized(); err != nil {
		return []string{}, err
	}

	if err := deviceEnumerator.AddMatchProperty("DEVTYPE", "disk"); err != nil {
		return []string{}, err
	}

	iterator, err := deviceEnumerator.DeviceIterator()
	if err != nil {
		return []string{}, err
	}

	disks := []string{}

	for {
		val, err := iterator.Next()
		if err != nil {
			if err != iter.FINISHED {
				return []string{}, err
			}
			break
		}
		device := val.(*udev.Device)
		disks = append(disks, device.Devnode())
	}

	return disks, nil
}

func GetPartitions(diskPath string) ([]string, error) {

}