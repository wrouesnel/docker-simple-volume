package volumequery

import (
	"bufio"
	"encoding/json"
	"os"
	"sort"

	"github.com/jochenvg/go-udev"

	"github.com/wrouesnel/docker-simple-disk/volumelabel"
	"gopkg.in/alecthomas/kingpin.v2"

	"errors"
	"fmt"
	"github.com/hashicorp/errwrap"
	"path"
	"path/filepath"

	"github.com/wrouesnel/go.log"
	"strings"
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
	errDiskNotFound                    = errors.New("given disk path did not resolve to a disk")
	errUdevDatabaseLookup              = errors.New("udev database snapshot failed")
	errBadGlobPattern                  = errors.New("bad glob pattern")
)

type DiskFailReason error

var (
	errUnknown                      = DiskFailReason(errors.New("disk status not known"))
	errBlankDisk                    = DiskFailReason(errors.New("disk is completely blank"))
	errHasAFilesystem               = DiskFailReason(errors.New("disk has no partitions but has a filesystem"))
	errHasPartitionTable            = DiskFailReason(errors.New("disk has no partitions but has a partition table"))
	errCouldNotFindLabelPartition   = DiskFailReason(errors.New("could not find label partition"))
	errCouldNotFindDataPartition    = DiskFailReason(errors.New("could not find data partition"))
	errFoundMultipleLabelPartitions = DiskFailReason(errors.New("found multiple label partitions after volume setup"))
	errFoundMultipleDataPartitions  = DiskFailReason(errors.New("found multiple data partitions after volume setup"))
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
	Encrypted bool `json:"encrypted"`
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
		Subsystems: make([]string, 0),
		Name:       make([]string, 0),
		Tag:        make([]string, 0),
		Properties: make(map[string]string),
		Attrs:      make(map[string]string),
	}
}

// selectionRuleForDevicePath generates the selection rule set which would
// query up a disk path.
func selectionRuleForDevicePath(devicePath string) []DeviceSelectionRule {
	return []DeviceSelectionRule{
		DeviceSelectionRule{
			Properties: map[string]string{
				"DEVNAME": devicePath,
			},
		},
	}
}

// deviceToSelectionRule converts a udev Device to a selection rule.
func deviceToSelectionRule(device *udev.Device) *DeviceSelectionRule {
	fixedRule := new(DeviceSelectionRule)

	// Currently looks like no way to actually get this?
	fixedRule.Name = []string{path.Base(device.Syspath())}

	fixedRule.Attrs = make(map[string]string)
	for attrName, _ := range device.Sysattrs() {
		fixedRule.Attrs[attrName] = device.SysattrValue(attrName)
	}
	fixedRule.Properties = device.Properties()
	fixedRule.Subsystems = []string{device.Subsystem()}

	fixedRule.Tag = []string{}
	for tagName, _ := range device.Tags() {
		fixedRule.Tag = append(fixedRule.Tag, tagName)
	}

	return fixedRule
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
func getDevicesByDevNode(selectionRules []DeviceSelectionRule) (map[string]*DeviceSelectionRule, error) {
	udevCtx := udev.Udev{}

	// Deduplicates the device paths we've already seen.
	devPaths := make(map[string]*DeviceSelectionRule)

	deviceEnumerator := udevCtx.NewEnumerate()
	// Only match initialized devices (global rule)
	if err := deviceEnumerator.AddMatchIsInitialized(); err != nil {
		return nil, err
	}

	devices, err := deviceEnumerator.Devices()
	if err != nil {
		return nil, errwrap.Wrap(errUdevDatabaseLookup, err)
	}

	// Filter the database snapshot by the selection rules from udev
	for _, rule := range selectionRules {
		var currentDevices []*udev.Device
		var nextDevices []*udev.Device

		// Filter mismatching subsystems
		currentDevices = devices[:] // Special: load the entire list
		nextDevices = make([]*udev.Device, 0, len(currentDevices))
		for _, device := range currentDevices {
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
				return nil, err
			}
			if deviceMatch {
				nextDevices = append(nextDevices, device)
			}
		}

		// Filter mismatching names
		currentDevices = nextDevices[:]
		nextDevices = make([]*udev.Device, 0, len(currentDevices))
		for _, device := range currentDevices {
			deviceMatch, err := func(device *udev.Device) (bool, error) {
				for _, name := range rule.Name {
					deviceName := filepath.Base(device.Syspath())
					matched, err := filepath.Match(name, deviceName)
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
				return nil, err
			}
			if deviceMatch {
				nextDevices = append(nextDevices, device)
			}
		}

		// Filter mismatching tags
		currentDevices = nextDevices[:]
		nextDevices = make([]*udev.Device, 0, len(currentDevices))
		for _, device := range currentDevices {
			deviceMatch, err := func(device *udev.Device) (bool, error) {
				// Each tag rule must match at least 1 device tag
				for _, tag := range rule.Tag {
					for deviceTag, _ := range device.Tags() {
						matched, err := filepath.Match(tag, deviceTag)
						if err != nil {
							return false, errwrap.Wrap(errBadGlobPattern, err)
						}
						// If the match failed, then this tag rule is not
						// satisfied, and so the device is not statisfied.
						if !matched {
							return false, nil
						}
					}
				}
				// Got through and matched all tags. Device matches.
				return true, nil
			}(device)
			// Fail on an error
			if err != nil {
				return nil, err
			}
			if deviceMatch {
				nextDevices = append(nextDevices, device)
			}
		}

		// Filter mismatching properties
		currentDevices = nextDevices[:]
		nextDevices = make([]*udev.Device, 0, len(currentDevices))
		for _, device := range currentDevices {
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
				return nil, err
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
				devPaths[device.Devnode()] = deviceToSelectionRule(device)
			}
		}
	}
	return devPaths, nil
}

// GetFullSelectionRulesForDevice queries a device by device path and returns a
// selection rule block which would uniquely match it. Mostly useful for
// simplectl to crosscheck rules.
func GetFullSelectionRuleForDevice(diskPath string) (*DeviceSelectionRule, error) {
	// This function uses targeted device selction rules
	devices, err := getDevicesByDevNode(selectionRuleForDevicePath(diskPath))
	if err != nil {
		return nil, err
	}

	if len(devices) > 1 {
		return nil, errGotMultipleDisksWhenExpectedOne
	}

	for _, device := range devices {
		return device, nil
	}

	return nil, errDiskNotFound
}

// GetDevicePaths returns a sorted list of disk devices which are matched by
// the DeviceSelectionRule
func GetDevicePaths(selectionRules []DeviceSelectionRule) ([]string, error) {
	devNodes := []string{}
	devices, err := getDevicesByDevNode(selectionRules)
	if err != nil {
		return []string{}, err
	}

	for devNode, _ := range devices {
		devNodes = append(devNodes, devNode)
	}

	// Sort the list
	sort.Strings(devNodes)

	return devNodes, nil
}

// GetPartitionDevicesFromDiskPath takes a disk path and returns the device
// path deduplicated list of partitions on the disk.
func GetPartitionDevicesFromDiskPath(diskPath string) (map[string]*DeviceSelectionRule, error) {
	// This function uses targeted device selction rules
	devices, err := getDevicesByDevNode(selectionRuleForDevicePath(diskPath))
	if err != nil {
		return nil, err
	}
	// Check only 1 disk was returned (doesn't make much sense otherwise so
	// we rule it out here explicitely.
	if len(devices) > 1 {
		return nil, errGotMultipleDisksWhenExpectedOne
	}

	partDiskVal := ""
	for _, device := range devices {
		partDiskVal = fmt.Sprintf("%s:%s",
			device.Properties["MAJOR"], device.Properties["MINOR"])
	}
	if partDiskVal == "" {
		return nil, errDiskNotFound
	}

	// Okay, query up the partitions
	partitionRules := []DeviceSelectionRule{
		DeviceSelectionRule{
			Properties: map[string]string{
				"ID_PART_ENTRY_DISK": partDiskVal,
				"DEVTYPE":            "partition",
			},
		},
	}

	partitionDevices, err := getDevicesByDevNode(partitionRules)

	return partitionDevices, nil
}

// GetDiskDeviceFromPartitionPath takes a partition path and tries to query the
// disk it is attached to by device number. It is guaranteed to only ever return
// the one device.
func GetDiskDeviceFromPartitionPath(partitionPath string) (map[string]*DeviceSelectionRule, error) {
	// This function uses targeted device selction rules
	partDevices, err := getDevicesByDevNode(selectionRuleForDevicePath(partitionPath))
	if err != nil {
		return nil, err
	}

	var major, minor string
	for _, partDev := range partDevices {
		diskTuple := strings.Split(partDev.Properties["ID_PART_ENTRY_DISK"], ":")

		// If found major minor and do not have agreement, error.
		if (major != "" && major != diskTuple[0]) || minor != "" && minor != diskTuple[1] {
			return nil, errGotMultipleDisksWhenExpectedOne
		}

		if major == "" && minor == "" {
			major = diskTuple[0]
			minor = diskTuple[1]
		}
	}

	// Query up the disk from the major/minor number
	diskRules := []DeviceSelectionRule{
		DeviceSelectionRule{
			Properties: map[string]string{
				"MAJOR": major,
				"MINOR": minor,
			},
		},
	}
	diskDevices, err := getDevicesByDevNode(diskRules)
	if len(diskDevices) == 0 {
		return nil, errDiskNotFound
	}

	if len(diskDevices) > 1 {
		return nil, errGotMultipleDisksWhenExpectedOne
	}

	return diskDevices, nil
}

// GetCandidateDisks returns all disks that simple might be able to use safely.
// A safe disk is either one which is already labelled as a simple disk, or
// one which is unpartitioned and does not appear to contain a filesystem or
// appear in the mount table.
func GetCandidateDisks(selectionRules []DeviceSelectionRule) (map[string]*DeviceSelectionRule, error) {

}

// GetInitializedDisks returns all disks on the current node which are
// initialized for simple.
func GetInitializedDisks(selectionRules []DeviceSelectionRule) (uninitializedDisks []string, initializedDisks []Disk, rerr error) {
	// Get all possible candidate devices
	disksPath, err := GetDevicePaths(selectionRules)
	if err != nil {
		rerr = err
		return
	}

	// Get the partitions of every candidate
	for _, diskPath := range disksPath {

	}
}

// CheckIfDiskIsInitialized takes a device path and determines if it is a
// simple disk. It returns the outcome of the assessment, a reason code if the
// assessment fails, and a failure code if the lookup fails.
func CheckIfDiskIsInitialized(diskPath string) (bool, DiskFailReason, error) {
	partDevices, err := GetPartitionDevicesFromDiskPath(diskPath)
	if err != nil {
		return false, errUnknown, err
	}

	if len(partDevices) == 0 {
		// Okay, no partitions. Does it have a filesystem?
		device, err := GetFullSelectionRuleForDevice(diskPath)
		if err != nil {
			return false, errUnknown, err
		}

		if _, found := device.Properties["ID_FS_USAGE"]; found {
			// Has a filesystem. Don't touch it.
			return false, errHasAFilesystem, nil
		} else if _, found := device.Properties["ID_PART_TABLE_TYPE"]; found {
			// Has a partition table. We wouldn't have create this, so don't
			// touch it.
			return false, errHasPartitionTable, nil
		}

		// No filesystems or partition tables - so just a blank disk.
		return false, errBlankDisk, nil
	}

	// Has some partitions. Is one a label partition
	labelDevice := ""
	dataDevice := ""
	for partPath, partDev := range partDevices {
		if partDev.Properties["ID_PART_ENTRY_NAME"] == SimpleMetadataLabel &&
			partDev.Properties["ID_PART_ENTRY_TYPE"] == SimpleMetadataUUID {
			if labelDevice != "" {
				return false, errFoundMultipleLabelPartitions, nil
			}
			labelDevice = partPath
			log.Debugln("Found simple label partition:", labelDevice)
		} else {
			if dataDevice == "" {
				dataDevice = partPath
				log.Debugln("Found simple data partition:", dataDevice)
			} else {
				return false, errFoundMultipleDataPartitions, nil
			}
		}
	}

	if labelDevice == "" {
		return false, errCouldNotFindLabelPartition, nil
	}

	if dataDevice == "" {
		return false, errCouldNotFindDataPartition, nil
	}

	// Disk is initialized and formatted properly.
	return true, nil, nil
}

// CheckIfDiskIsBlankCandidate ensures the disk has no filesystems, no partition
// table, and can be safely recruited as a simple disk.
func CheckIfDiskIsBlankCandidate(diskPath string) (bool, error) {
	isInitialized, failReason, err := CheckIfDiskIsInitialized(diskPath)
	if err != nil {
		return false, err
	}
	// If disk is already initialized, its definitely not blank.
	if isInitialized {
		return false, nil
	}

	// If the disk isn't an initialized disk because it's a blank disk, then its
	// a candidate to be initialized.
	if failReason == errBlankDisk {
		return true, nil
	}

	return false, nil
}
