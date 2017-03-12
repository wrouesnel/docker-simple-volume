package volumequery

import (
	"sort"
	"encoding/json"
	"bufio"
	"os"

	"github.com/jochenvg/go-udev"

	"github.com/wrouesnel/docker-simple-disk/volumelabel"
	"gopkg.in/alecthomas/kingpin.v2"

	//linuxproc "github.com/c9s/goprocinfo/linux"
	//"github.com/wrouesnel/go.log"
	"errors"
	"fmt"
	"path"
	"github.com/hashicorp/errwrap"
	"path/filepath"
)

const SimpleMetadataLabel string = "simple-metadata"
const SimpleMetadataUUID string = "903b0d2d-812e-4029-89fa-a905b9cd80c1"

type NamingType string

const (
	NamingNumeric NamingType = "numeric"
	NamingUUID    NamingType = "uuid"
)

const (
	VolumeLabelVersion int = 1
)

const (
	// Where to read mountpoint info from.
	ProcMounts string = "/proc/mounts"
)

var (
	errGotMultipleDisksWhenExpectedOne = errors.New("got multiple disk devices from a query when only 1 was expected")
	errDiskNotFound = errors.New("given disk path did not resolve to a disk")
	errUdevDatabaseLookup = errors.New("udev database snapshot failed")
	errBadGlobPattern = errors.New("bad glob pattern")
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
	// Version of the label schema
	Version int
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

// getDevicesByDevNode takes a list of udev selection rules while will be
// applied individually and the list of devices appended and returned. The
// final list is deduplicated on the basis of DevPath (i.e. /dev/<device>)
// The list is returned as a list of *udev.Device nodes.
//
// Usage: multiple rules can have varying levels of specificity - devices
// matched by a less specific rule are deduplicated on the basis of device path.
//
// Performance note: udev is weird about rule application - adding a match for
// properties is an OR operation, not an AND which doesn't suite our purposes
// at all, so this function just dumps the DB and implements it's own filtering.
// This implements glob matching so it should broadly follow what's possible.
func getDevicesByDevNode(selectionRules []DeviceSelectionRule) ([]*udev.Device, error) {
	udevCtx := udev.Udev{}

	// Deduplicates the device paths we've already seen.
	devPaths := make(map[string]struct{})

	// Returns the list of devices foudn through the list of selection rules
	returnedDevices := []*udev.Device{}

	deviceEnumerator := udevCtx.NewEnumerate()
	// Only match initialized devices (global rule)
	if err := deviceEnumerator.AddMatchIsInitialized(); err != nil {
		return []*udev.Device{}, err
	}

	devices, err := deviceEnumerator.Devices()
	if err != nil {
		return []*udev.Device{}, errwrap.Wrap(errUdevDatabaseLookup, err)
	}

	// Filter the database snapshot by the selection rules from udev
	for _, rule := range selectionRules {
		var currentDevices []*udev.Device
		var nextDevices []*udev.Device

		// Filter mismatching subsystems
		currentDevices = devices[:]	// Special: load the entire list
		nextDevices = make([]*udev.Device, 0, len(currentDevices))
		for _, device := range devices {
			deviceMatch, err := func(device *udev.Device) (bool, error) {
				for _, subsystem := range rule.Subsystems {
					matched, err := filepath.Match(subsystem, device.Subsystem())
					if err != nil {
						return false, errwrap.Wrap(errBadGlobPattern, err)
					}
					// Any match failure cancels the device out of the set
					if !matched {
						return false, nil
					}
				}
				// Got through rules with no failed matches.
				return true, nil
			}(device)
			// Fail on an error
			if err != nil {
				return []*udev.Device{}, err
			}
			if deviceMatch {
				nextDevices = append(nextDevices, device)
			}
		}

		// Filter mismatching names
		currentDevices = nextDevices[:]
		nextDevices = make([]*udev.Device, 0, len(currentDevices))
		for _, device := range devices {
			deviceMatch, err := func(device *udev.Device) (bool, error) {
				for _, subsystem := range rule.Name {
					matched, err := filepath.Match(subsystem, filepath.Base(device.Syspath()))
					if err != nil {
						return false, errwrap.Wrap(errBadGlobPattern, err)
					}
					// Any match failure cancels the device out of the set
					if !matched {
						return false, nil
					}
				}
				// Got through rules with no failed matches.
				return true, nil
			}(device)
			// Fail on an error
			if err != nil {
				return []*udev.Device{}, err
			}
			if deviceMatch {
				nextDevices = append(nextDevices, device)
			}
		}

		// Filter mismatching tags
		currentDevices = nextDevices[:]
		nextDevices = make([]*udev.Device, 0, len(currentDevices))
		for _, device := range devices {
			deviceMatch, err := func(device *udev.Device) (bool,error) {
				// Each tag rule must match *any* tag in the device tag set
				for deviceTag, _ := range device.Tags() {
					tagMatch := false
					for _, tag := range rule.Tag {
						matched, err := filepath.Match(tag, deviceTag)
						if err != nil {
							return false, errwrap.Wrap(errBadGlobPattern, err)
						}
						// Matched - can stop checking
						if matched {
							tagMatch = true
							break
						}
					}
					// A deviceTag did not match any rules, so device is not
					// matched.
					if !tagMatch {
						return false, nil
					}
				}
				// Got through and matched all tags. Device matches.
				return true, nil
			}(device)
			// Fail on an error
			if err != nil {
				return []*udev.Device{}, err
			}
			if deviceMatch {
				nextDevices = append(nextDevices, device)
			}
		}

		// Filter mismatching properties
		currentDevices = nextDevices[:]
		nextDevices = make([]*udev.Device, 0, len(currentDevices))
		for _, device := range devices {
			// Use this closure to handle the complex rules.
			deviceMatch, err := func(device *udev.Device) (bool, error) {
				// Every key glob must match at least a key.
				// Then it's value glob must match the given value.
				// This is probably the most inefficient search space.
				for keyGlob, valueGlob := range rule.Properties {
					globMatch := false
					for deviceKey, deviceValue := range device.Properties() {
						keyMatched, err := filepath.Match(keyGlob, deviceKey)
						if err != nil {
							return false, errwrap.Wrap(errBadGlobPattern, err)
						}
						// Didn't match this key - try others
						if !keyMatched {
							continue
						}
						// Key glob matched. Does the value glob match its value?
						valueMatched, err := filepath.Match(valueGlob, deviceValue)
						if err != nil {
							return false, errwrap.Wrap(errBadGlobPattern, err)
						}
						// Matched on the value, so can break loop and mark
						// glob success
						if valueMatched {
							globMatch = true
							break
						}
					}

					if !globMatch {
						// Glob match failed for this property rule, so it
						// cancels the device out of the set.
						return false, nil
					}
				}
				// Got through every check and the device still matched.
				return true, nil
			}(device)
			// Fail on an error
			if err != nil {
				return []*udev.Device{}, err
			}
			if deviceMatch {
				nextDevices = append(nextDevices, device)
			}
		}
		currentDevices = nextDevices[:]

		// currentDevices is now the remaining devices which survived our rules.
		// Figure out if we have new devices.
		for _, device := range currentDevices {
			if _, found := devPaths[device.Devnode()]; !found {
				devPaths[device.Devnode()] = struct{}{}
				returnedDevices = append(returnedDevices, device)
			}
		}
	}

	return returnedDevices, nil
}

// GetFullSelectionRulesForDevice queries a device by device path and returns a
// selection rule block which would uniquely match it. Mostly useful for
// simplectl to crosscheck rules.
func GetFullSelectionRulesForDevice(diskPath string) ([]*DeviceSelectionRule, error) {
	// This function uses targeted device selction rules
	diskRules := []DeviceSelectionRule{
		DeviceSelectionRule{
			Properties: map[string]string{
				"DEVNAME": diskPath,
			},
		},
	}

	devices, err := getDevicesByDevNode(diskRules)
	if err != nil {
		return nil, err
	}

	if len(devices) == 0 {
		return nil, errDiskNotFound
	}

	if len(devices) > 1 {
		return nil, errGotMultipleDisksWhenExpectedOne
	}
	device := devices[0]

	fixedRules := new(DeviceSelectionRule)

	// Currently looks like no way to actually get this?
	fixedRules.Name = []string{path.Base(device.Syspath())}

	fixedRules.Attrs = make(map[string]string)
	for attrName, _ := range device.Sysattrs() {
		fixedRules.Attrs[attrName] = device.SysattrValue(attrName)
	}
	fixedRules.Properties = device.Properties()
	fixedRules.Subsystems = []string{device.Subsystem()}

	fixedRules.Tag = []string{}
	for tagName, _ := range device.Tags() {
		fixedRules.Tag = append(fixedRules.Tag, tagName)
	}

	return []*DeviceSelectionRule{fixedRules}, nil
}

// GetCandidateDisks returns a sorted list of disk devices which are matched by
// the DeviceSelectionRule
func GetCandidateDisks(selectionRules []DeviceSelectionRule) ([]string, error) {
	devNodes := []string{}
	devices, err := getDevicesByDevNode(selectionRules)
	if err != nil {
		return []string{}, err
	}

	for _, d := range devices {
		devNodes = append(devNodes, d.Devnode())
	}

	// Sort the list
	sort.Strings(devNodes)

	return devNodes, nil
}

// GetPartitionDevicesFromDiskPath takes a disk path and returns a list of
// partition devices available on the disk.
func GetPartitionDevicesFromDiskPath(diskPath string) ([]string, error) {
	// This function uses targeted device selction rules
	diskRules := []DeviceSelectionRule{
		DeviceSelectionRule{
			Properties: map[string]string{
				"DEVNAME": diskPath,
			},
		},
	}

	devices, err := getDevicesByDevNode(diskRules)
	if err != nil {
		return []string{}, err
	}
	// Check only 1 disk was returned (doesn't make much sense otherwise so
	// we rule it out here explicitely.
	if len(devices) > 1 {
		return []string{}, errGotMultipleDisksWhenExpectedOne
	}
	// No devices - raise an error, probably not what was expected.
	if len(devices) == 0 {
		return []string{}, errDiskNotFound
	}
	diskMajor := devices[0].Devnum().Major()
	diskMinor := devices[0].Devnum().Minor()
	partDiskVal := fmt.Sprintf("%d:%d", diskMajor, diskMinor)

	// Okay, query up
	partitionRules := []DeviceSelectionRule{
		DeviceSelectionRule{
			Properties: map[string]string{
				"ID_PART_ENTRY_DISK": partDiskVal,
				"DEVTYPE": "partition",
			},
		},
	}

	partitionDevices, err := getDevicesByDevNode(partitionRules)

	matchedPartitions := []string{}
	for _, part := range partitionDevices {
		matchedPartitions = append(matchedPartitions, part.Devnode())
	}

	// Sort the list
	sort.Strings(matchedPartitions)

	return matchedPartitions, nil
}

// GetInitializedDisks returns all disks on the current node which are
// initialized.
//func GetInitializedDisks() []Disk {
//
//}

// GetCandidateDisks returns all disks on the current node which *could* be used
// and filters the list of possible disks on the basis of whether they are
// presently mounted and marked exclusive.
// selectionRules : should be set to the global device candidate rule to use for
// 					setting
//func GetAvailableCandidateDisks(selectionRules []DeviceSelectionRule) []string {
//	// Read the mounts
//	mounts, err := linuxproc.ReadMounts(ProcMounts)
//	if err != nil {
//		log.Errorln("Error reading mounts - no candidate devices will be allowed:", err)
//		return []string{}
//	}
//	// Generate the map of disks with currently mounted partitions
//	mountMap := make(map[string]struct{})
//	for _, mnt := range mounts.Mounts {
//		mountMap[mnt.Device] = struct{}{}
//	}
//	// Read the list of possible disks
//	disks, err := GetDevicesByDevNode(selectionRules)
//	if err != nil {
//		return []string{}
//	}
//	// Remove disks which have partitions currently mounted from them
//	for _, disk := range disks {
//		// Ugly - feels like there should be a better way, but udev doesn't
//		// actually get much more specific
//
//	}
//
//
//
//
//}