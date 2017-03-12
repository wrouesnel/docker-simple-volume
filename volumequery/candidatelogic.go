// Contains the functions which implement the actual simple candidate disk logic
// as opposed to generic-ish udev query functions.

package volumequery

import (
	"github.com/wrouesnel/go.log"
)

// GetCandidateDisks returns all disks that simple might be able to use safely.
// A safe disk is either one which is already labelled as a simple disk, or
// one which is unpartitioned and does not appear to contain a filesystem or
// appear in the mount table.
func GetCandidateDisks(selectionRules []DeviceSelectionRule) (initialized []string, uninitialized []string, rejected []string, rerr error) {
	diskPaths, err := GetDevicePaths(selectionRules)
	if err != nil {
		rerr = err
		return
	}
	for _, diskPath := range diskPaths {
		isInitialized, failReason, err := CheckIfDiskIsInitialized(diskPath)
		if err != nil {
			rerr = err
			return
		}
		// Sort the disk based on what it is...
		if isInitialized {
			initialized = append(initialized, diskPath)
		} else if IsBlankDisk(isInitialized, failReason) {
			uninitialized = append(uninitialized, diskPath)
		} else {
			rejected = append(rejected, diskPath)
		}
	}
	// All good! We have our candidates!
	return
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

// IsBlankDisk converts an isInitialized/failReason pair into a check if the
// disk is blank.
func IsBlankDisk(isInitialized bool, failReason DiskFailReason) bool {
	// If disk is already initialized, its definitely not blank.
	if isInitialized {
		return false
	}

	// If the disk isn't an initialized disk because it's a blank disk, then its
	// a candidate to be initialized.
	if failReason == errBlankDisk {
		return true
	}

	return false
}

// CheckIfDiskIsBlankCandidate ensures the disk has no filesystems, no partition
// table, and can be safely recruited as a simple disk. Internally it calls
// CheckIfDiskIsInitialized - if you need to do both checks, then it's better to
// use IsBlankDisk which just does the response code parsing.
func CheckIfDiskIsBlankCandidate(diskPath string) (bool, error) {
	isInitialized, failReason, err := CheckIfDiskIsInitialized(diskPath)
	if err != nil {
		return false, err
	}

	return IsBlankDisk(isInitialized, failReason), nil
}
