package volumequery

import (
	"os"
	"strconv"

	"github.com/coreos/go-systemd/util"
	"github.com/wrouesnel/docker-simple-disk/volumeaccess"
)

// FilterByVolumeQuery takes a list of initialized devices and filters the list
// for devices who's labels or properties match the supplied volume query.
//func FilterByVolumeQuery(query *VolumeQuery, diskPaths []string) (diskPath []string, error) {
//	for _, diskPath := range diskPaths {
//		labelPath, dataPath, err := GetDiskLabelAndVolumePath(diskPath)
//		if err != nil {
//			return
//		}
//	}
//
//}

// VolumeQueryMatch checks if a given volume query would match the device at
// the given path. Does not check for initialization or exclusive access
// constraints.
func VolumeQueryMatch(query *VolumeQuery, labelPath string, dataPath string) (bool, error) {
	label, err := DeserializeVolumeLabel(labelPath)
	if err != nil {
		return false, err
	}

	// By definition, any query for a non-initialized device should fail if
	// gets into this function. But - we probably initialized the volume before
	// we got here - so ignore this field here.

	// query.Initialized

	// Check query parameters which are determined from the labels first.
	if query.OwnHostname {
		// Try and get this machine's hostname
		ourHostname, err := os.Hostname()
		if err != nil {
			// Can't get hostname, but must match hostname == would always fail
			return false, err
		}
		if label.Hostname != ourHostname {
			return false, nil
		}
	}

	if query.OwnMachineId {
		// Try and get the machine ID
		ourMachineId, err := util.GetMachineID()
		if err != nil {
			// Can't get machineId, but must match machineId == would always fail
			return false, err
		}
		if label.MachineId != ourMachineId {
			return false, nil
		}
	}

	if query.Label != "" {
		// TODO: cross-match partition label
		if query.Label != label.Label {
			return false, nil
		}
	}

	// label.Numbering has no query relevance

	// Open the device into a context so we can inspect filesystems/size/etc
	var dataCtx volumeaccess.VolumeContext
	if query.EncryptionKey != "" {
		if !label.Encrypted {
			// label says device is not encrypted. It could be corrupted, but
			// someone would have to do this intentionally - so assume its a
			// problem and just fail.
			return false, nil
		}
		var err error
		dataCtx, err = volumeaccess.OpenEncryptedDevice(query.EncryptionKey, dataPath)
		if err != nil {
			// Encryption key does not unlock the encrypted volume
			return false, nil
		}
	} else {
		dataCtx, err = volumeaccess.OpenDevice(dataPath)
		if err != nil {
			// Encryption key does not unlock the encrypted volume
			return false, nil
		}
	}
	defer dataCtx.Close()

	// Have to query up the partition now to match these rules.
	rule, err := GetFullSelectionRuleForDevice(dataCtx.GetDevicePath())
	if err != nil {
		return false, err
	}

	sizeStr, found := rule.Attrs["size"]
	if !found {
		return false, nil
	}

	deviceSize, err := strconv.ParseUint(sizeStr, 10, 64)
	if err != nil {
		// Might be badly formatted - don't care. We can't match with it,
		// so fail it.
		return false, nil
	}

	if query.MinimumSizeBytes > 0 {
		if deviceSize < query.MinimumSizeBytes {
			return false, nil
		}
	}

	if query.MaximumSizeBytes > 0 {
		if deviceSize > query.MaximumSizeBytes {
			return false, nil
		}
	}

	if query.Filesystem != "" {
		deviceFs, found := rule.Properties["ID_FS_TYPE"]
		if !found {
			// Can't determine type - can't match on it - fail this device.
			return false, nil
		}
		if deviceFs != query.Filesystem {
			// Fail - filesystem does not match.
			return false, nil
		}
	}

	// All single-device constraints are satisifed by this query.
	return true, nil
}