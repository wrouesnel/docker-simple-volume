package volumequery

import (
	//linuxproc "github.com/c9s/goprocinfo/linux"
	//"github.com/wrouesnel/go.log"
)

// GetAvailableCandidateDisks returns all disks on the current node which
// *could* be used and filters the list of possible disks on the basis of
// whether they are presently mounted and marked exclusive.
// selectionRules : should be set to the global device candidate rule to use for
// 					setting
//func GetAvailableCandidateDisks(selectionRules []DeviceSelectionRule) []string {
//	// Read the mounts
//	mounts, err := linuxproc.ReadMounts(ProcMounts)
//	if err != nil {
//		log.Errorln("Error reading mounts - no candidate devices will be allowed:", err)
//		return []string{}
//	}
//	// Figure out which disks are already mounted.
//	inUseDisks := make(map[string]*DeviceSelectionRule)
//	for _, mnt := range mounts.Mounts {
//		// Query the partition.
//		diskDevices, err := GetDiskDeviceFromPartitionPath(mnt.Device)
//		// Skip this mount but proceed.
//		if err != nil {
//			if err == errGotMultipleDisksWhenExpectedOne {
//				log.Warnln("Found multiple parent disks for mounted device:", mnt.Device)
//			} else if err == errDiskNotFound {
//				log.Warnln("Could not find a parent disk for mounted device:", err)
//			} else {
//				log.Errorln("Unexpected error when querying parent disks:", err)
//			}
//			continue
//		}
//
//		for k, v := range diskDevices {
//			if _, found := inUseDisks[k]; found {
//				log.Debugln("Disk already detected as in-use:", k)
//			} else {
//				inUseDisks[k] = v
//			}
//		}
//	}
//
//
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
//}