package volumequery

import (
	"sort"
	"github.com/jkeiser/iter"
	"github.com/jochenvg/go-udev"
	"encoding/json"
	"bufio"
	"os"
	"github.com/wrouesnel/docker-simple-disk/volumelabel"
	"gopkg.in/alecthomas/kingpin.v2"
	"strings"
)

const SimpleMetadataLabel string = "simple-metadata"
const SimpleMetadataUUID string = "903b0d2d-812e-4029-89fa-a905b9cd80c1"

type NamingType string

const (
	NamingNumeric NamingType = "numeric"
	NamingUUID    NamingType = "uuid"
)

// Specifies a volume query (this is a mash-up of query and create parameters
// in reality.
type VolumeQuery struct {
	// Label of disk to search for
	Label string `volumelabel:"label"`

	// Hostname disk should be associated with
	OwnHostname bool `volumelabel:"own-hostname"`
	// MachineID the disk should be associated with
	OwnMachineId string `volumelabel:"own-machine-id"`

	// Should the disk have been initialized by the filesystem
	Initialized bool `volumelabel:"initialized"`
	// Should the disk be marked as exclusive use?
	Exclusive bool `volumelabel:"exclusive"`
	// Should the disk be placed in a subdirectory and dynamically updated
	// as matching disks are added/removed
	DynamicMounts bool `volumelabel:"dynamic-mounts"`
	// Should disk numbering fields be respected from the label?
	PersistNumbering bool `volumelabel:"persist-numbering"`

	// Basename is the prefix assigned to mounted disks under the volume.
	Basename string `volumelabel:"basename"`
	// Naming style to use for disk mounts - numeric (incremented numbers)
	// or uuid (what it sounds like).
	NamingStyle NamingType `volumelabel:"naming-style"`

	// Minimum disk size to be considered valid.
	MinimumSizeBytes uint64 `volumelabel:"min-size"`
	// Maximum disk size to be considered valid.
	MaximumSizeBytes uint64 `volumelabel:"max-size"`

	// Minimum number of disks which must match before returning
	MinDisks int32 `volumelabel:"min-disks"`
	// Maximum number of disks which can be returned by the match
	MaxDisks int32 `volumelabel:"max-disks"`

	// Filesystem which will be created or found
	Filesystem string `volumelabel:"filesystem"`

	// Encryption Key - if specified requires a volume be encrypted with the
	// given key.
	EncryptionKey string `volumelabel:"encryption-passphrase"`
	// LUKS cipher to be used if creating a volume. If a passphrase is
	// specified then uses the LUKS default.
	EncryptionCipher string `volumelabel:"encryption-hash"`
	// LUKS key size.
	EncryptionKeySize int `volumelabel:"encryption-key-size"`
	// LUKS hash function
	EncryptionHash string `volumelabel:"encryption-hash"`
}

// VolumeQueryValue implements flag parsing for VolumeQuery's
type VolumeQueryValue VolumeQuery

func (this *VolumeQueryValue) Set(value string) error {
	return volumelabel.UnmarshalVolumeLabel(value, this)
}

func (this *VolumeQueryValue) String() string {
	output, err := volumelabel.MarshalVolumeLabel(this)
	if err != nil {
		return ""
	}
	return output
}

// VolumeQueryVar implements a var mapped for reading VolumeQuery's with
// kingpin from the command line./
func VolumeQueryVar(settings kingpin.Settings, target *VolumeQuery) {
	settings.SetValue((*VolumeQueryValue)(target))
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
	// Disk was created as an encrypted volume
	Encrypted bool	`json:"encrypted"`
	// Extra metadata
	Metadata map[string]string `json:"metadata"`
}

// Serializes the label to it's null-terminated JSON form
func SerializeVolumeLabel(label *VolumeLabel) ([]byte, error) {
	serialized, err := json.Marshal(label)
	if err != nil {
		return []byte{}, err
	}
	serialized = append(serialized, byte(0))
	return serialized, nil
}

// DeserializeVolumeLabel reads a given path for a null terminated VolumeLabel
func DeserializeVolumeLabel(path string) (VolumeLabel, error) {
	f, err := os.Open(path)
	if err != nil {
		return VolumeLabel{}, err
	}

	rdr := bufio.NewReader(f)
	rawData, err := rdr.ReadBytes(byte(0))
	if err != nil {
		return VolumeLabel{}, err
	}

	volLabel := VolumeLabel{}

	if err := json.Unmarshal(rawData, &volLabel); err != nil {
		return VolumeLabel{}, err
	}

	return volLabel, nil
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

func (this *DeviceSelectionRule) Copy() DeviceSelectionRule {
	newRule := NewDeviceSelectionRule()

	for k, v := range this.Properties {
		newRule.Properties[k] = v
	}

	for k, v := range this.Attrs {
		newRule.Attrs[k] = v
	}

	return newRule
}

func NewDeviceSelectionRule() DeviceSelectionRule {
	return DeviceSelectionRule{
		Subsystems : make([]string, 0),
		Name : make([]string, 0),
		Tag : make([]string, 0),
		Properties: make(map[string]string),
		Attrs : make(map[string]string),
	}
}

// GetDevicesByDevNode takes a list of udev selection rules while will be
// applied individually and the list of devices appended and returned. The
// final list is deduplicated on the basis of DevPath (i.e. /dev/<device>)
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

// GetPartitionDevicesFromDiskPath takes a disk path and returns a list of
// partition devices available on the disk. Due to the limitations of what
// udev typically records, this currently is simply a prefix match against
// *all* the partitions on the system.
func GetPartitionDevicesFromDiskPath(diskPath string) ([]string, error) {
	// This function uses targeted device selction rules
	rules := []DeviceSelectionRule{
		DeviceSelectionRule{
			Properties: map[string]string{
				"DEVTYPE": "partition",
			},
		},
	}

	partitions, err := GetDevicesByDevNode(rules)
	if err != nil {
		return []string{}, err
	}

	matchedPartitions := []string{}
	for _, part := range partitions {
		if strings.HasPrefix(part, diskPath) {
			matchedPartitions = append(matchedPartitions, part)
		}
	}

	// Do the actual query
	return matchedPartitions, nil
}

// GetInitializedDisks returns all disks on the current node which are
// initialized.
//func GetInitializedDisks() []Disk {
//
//}

// GetCandidateDisks returns all disks on the current node which *could* be used
// Filters the list of possible disks on the basis of whether or not they are
// presently mounted, which removed
//func GetCandidateDisks() []string {
//
//}