package volumequery

import (
	"github.com/wrouesnel/docker-simple-disk/volumelabel"
	"github.com/jkeiser/iter"
	"github.com/jochenvg/go-udev"
	"sort"
	"errors"
)

type NamingType string

const (
	NamingNumeric NamingType = "numeric"
	NamingUUID    NamingType = "uuid"
)

// Specifies a volume query (this is a mash-up of query and create parameters
// in reality.
type VolumeQuery struct {
	// Label of disk to search for
	Label string	`volumelabel:"label"`

	// Hostname disk should be associated with
	OwnHostname  bool	`volumelabel:"own-hostname"`
	// MachineID the disk should be associated with
	OwnMachineId string `volumelabel:"own-machine-id"`

	// Should the disk have been initialized by the filesystem
	Initialized      bool	`volumelabel:"initialized"`
	// Should the disk be marked as exclusive use?
	Exclusive        bool	`volumelabel:"exclusive"`
	// Should the disk be placed in a subdirectory and dynamically updated
	// as matching disks are added/removed
	DynamicMounts    bool	`volumelabel:"dynamic-mounts"`
	// Should disk numbering fields be respected from the label?
	PersistNumbering bool	`volumelabel:"persist-numbering"`

	// Basename is the prefix assigned to mounted disks under the volume.
	Basename    string		`volumelabel:"basename"`
	// Naming style to use for disk mounts - numeric (incremented numbers)
	// or uuid (what it sounds like).
	NamingStyle NamingType	`volumelabel:"naming-style"`

	// Minimum disk size to be considered valid.
	MinimumSizeBytes uint64	`volumelabel:"min-size"`
	// Maximum disk size to be considered valid.
	MaximumSizeBytes uint64	`volumelabel:"max-size"`

	// Minimum number of disks which must match before returning
	MinDisks int32	`volumelabel:"min-disks"`
	// Maximum number of disks which can be returned by the match
	MaxDisks int32	`volumelabel:"max-disks"`

	// Filesystem which will be created / or found
	Filesystem string	`volumelabel:"filesytem"`

	// Additional key-value metadata which must match.
	Metadata map[string]string
}

func (this VolumeQuery) MarshalVolumeLabel() (string, error) {
	return "", errors.New("Not implemented")
}

func (this *VolumeQuery) UnmarshalVolumeLabel(l string) error {
	return errors.New("Not implemented")
}

// Struct representing labelled data (output as JSON)
type VolumeLabel struct {
	// Hostname this disk was last initialized on
	Hostname string `json:"hostname"`
	// Machine ID this disk was last initialized on, if available
	MachineId string `json:"machine_id"`
	// Label of this disk (should match partition label)
	Label string `json:"label"`
	// Last numbering assignment this disk had for the current label
	Numbering string `json:"numbering"`
	// Extra metadata
	Metadata map[string]string `json:"metadata"`
}

// Struct representing a disk which is able to be used by the plugin
type Disk struct {
	Label         VolumeLabel
	PartitionPath string
}

type DeviceSelectionRule struct {
	Subsystems []string
	Name       []string
	Tag        []string
	Properties map[string]string
	Attrs      map[string]string
}

// Takes a list of udev selection rules while will be applied individually and
// the list of devices appended and returned. The final list is deduplicated
// on the basis of DevPath (i.e. /dev/<device>)
func GetDevicesByDevNode(selectionRules []DeviceSelectionRule) ([]string, error) {
	udevCtx := udev.Udev{}

	devPaths := make(map[string]interface{})

	for _, rule := range selectionRules {
		deviceEnumerator := udevCtx.NewEnumerate()

		// Only match initialized devices
		if err := deviceEnumerator.AddMatchIsInitialized(); err != nil {
			return []string{}, err
		}

		for _, subsystem := range rule.Subsystems {
			if err := deviceEnumerator.AddMatchSubsystem(subsystem); err != nil {
				return []string{}, err
			}
		}

		for _, name := range rule.Name {
			if err := deviceEnumerator.AddMatchSysname(name); err != nil {
				return []string{}, err
			}
		}

		for _, tag := range rule.Tag {
			if err := deviceEnumerator.AddMatchTag(tag); err != nil {
				return []string{}, err
			}
		}

		for key, val := range rule.Properties {
			if err := deviceEnumerator.AddMatchProperty(key, val); err != nil {
				return []string{}, err
			}
		}

		for key, val := range rule.Attrs {
			if err := deviceEnumerator.AddMatchSysattr(key, val); err != nil {
				return []string{}, err
			}
		}

		// Do the actual query
		iterator, err := deviceEnumerator.DeviceIterator()
		if err != nil {
			return []string{}, err
		}

		for {
			val, err := iterator.Next()
			if err != nil {
				if err != iter.FINISHED {
					return []string{}, err
				}
				break
			}
			device := val.(*udev.Device)

			devPaths[device.Devnode()] = nil
		}
	}

	ret := []string{}
	for key, _ := range devPaths {
		ret = append(ret, key)
	}

	// Sort the array
	sort.Strings(ret)

	return ret, nil
}

// Takes a dispath and returns a list of partition devices available on the
// disk. This is eschewing manually reading GPT, which would be inefficient
// since we already bind libudev.
func GetPartitionDevicesFromDiskPath(diskPath string) ([]string, error) {
	udevCtx := udev.Udev{}

	devPaths := make(map[string]interface{})

	deviceEnumerator := udevCtx.NewEnumerate()

	// Only match initialized devices
	if err := deviceEnumerator.AddMatchIsInitialized(); err != nil {
		return []string{}, err
	}
}
