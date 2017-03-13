package volumesetup

import (
	"os"
	"fmt"
	"errors"
	"strings"

	"github.com/hashicorp/errwrap"
	"github.com/wrouesnel/go.sysutil/executil"
	"github.com/wrouesnel/go.log"

	"github.com/wrouesnel/docker-simple-disk/volumequery"
	"github.com/wrouesnel/docker-simple-disk/volumeaccess"
)

var (
	errPartitioningFailed = errors.New("failed to partition disk")
	errPartProbeFailed = errors.New("informing kernel of partition update failed")
	errDiskDidNotInitialize = errors.New("disk did not return as initialized after it should have")
	errCouldNotWriteVolumeLabel = errors.New("failed to write volumelabel")
	errCryptSetupFailed = errors.New("error setting up encrypted device")
	errFilesystemCreationFailed = errors.New("error creating filesystem")
)

const (
	PartitionLabelSize int = 1
	PartitionLabelInitialOffset int = 1
)

// Initialize a block device as a docker-simple-disk device based on a volume
// query. This function will forcibly overwrite any partition table already
// present.
func InitializeBlockDevice(blockDevice string, inputQuery volumequery.VolumeQuery, hostname string, machineId string) error {

	// Partition index we're aligning too
	partIdx := 1

	// Initial sgdisk partition label is *always* the metadata
	partitionOpts := []string{
		"-o",
		"-n",
		fmt.Sprintf("%d:%dM:%dM", partIdx, PartitionLabelInitialOffset, PartitionLabelInitialOffset + PartitionLabelSize),
		"-t",
		fmt.Sprintf("%d:%s", partIdx, volumequery.SimpleMetadataUUID),
		"-c",
		fmt.Sprintf("%d:%s", partIdx, volumequery.SimpleMetadataLabel),
	}

	// Parse the volume query into options which affect partitioning
	// TODO: do we need to set type and do some magic to line it up?
	partitionLabel := inputQuery.Label
	partIdx++
	partitionOpts = append(partitionOpts,
		"-n",
		fmt.Sprintf("%d:0:0", partIdx),
	)

	if partitionLabel != "" {
		partitionOpts = append(partitionOpts, "-c", fmt.Sprintf("%d:%s",partIdx, partitionLabel))
	}
	partitionOpts = append(partitionOpts, blockDevice)

	log.Infoln("Partitioning device")
	log.Debugln("Partitioning with commandline: sgdisk", strings.Join(partitionOpts, " "))
	if err := executil.CheckExec("sgdisk", partitionOpts...); err != nil {
		return errwrap.Wrap(errPartitioningFailed, err)
	}

	log.Infoln("Updating kernel with new device partitions")
	if err := executil.CheckExec("partprobe", blockDevice); err != nil {
		return errwrap.Wrap(errPartProbeFailed, err)
	}

	// Generate a VolumeLabel structure from the VolumeQuery
	isEncrypted := false
	if inputQuery.EncryptionKey != "" {
		isEncrypted = true
	}

	label := volumequery.VolumeLabel{
		Version: volumequery.VolumeLabelVersion,
		Hostname: hostname,
		MachineId: machineId,
		Label: partitionLabel,
		Numbering: "",
		Encrypted: isEncrypted,
		Metadata: make(map[string]string),
	}

	log.Infoln("Checking new device is initialized")
	labelDevice, dataDevice, err := volumequery.GetDiskLabelAndVolumePath(blockDevice)
	if err != nil {
		return errwrap.Wrap(errDiskDidNotInitialize, err)
	}
	log.Infoln("Disk Device", blockDevice, "has label device", labelDevice, "and data device", dataDevice)

	log.Infoln("Writing label content to:", labelDevice)
	log.Debugln("Serializing volume label")
	labelBytes, err := volumequery.SerializeVolumeLabel(&label)
	if err != nil {
		return errwrap.Wrap(errCouldNotWriteVolumeLabel, err)
	}
	log.Debugln("Writing label")
	if err := WriteAndSyncExistingFile(labelDevice, labelBytes, os.FileMode(0600)); err != nil {
		return errwrap.Wrap(errCouldNotWriteVolumeLabel, err)
	}

	log.Infoln("Setting up data volume")
	fsDevice := dataDevice

	// Is this an encrypted volume?
	if isEncrypted {
		log.Infoln("Setting up encrypted volume")
		cryptOpts := []string{
			"-v",
			"--force-password",
			"luksFormat",
		}

		if inputQuery.EncryptionCipher != "" {
			cryptOpts = append(cryptOpts, "-c", inputQuery.EncryptionCipher)
		}

		if inputQuery.EncryptionKeySize != 0 {
			cryptOpts = append(cryptOpts, "-s", fmt.Sprintf("%d", inputQuery.EncryptionKeySize))
		}

		if inputQuery.EncryptionHash != "" {
			cryptOpts = append(cryptOpts, "-h", inputQuery.EncryptionHash)
		}

		cryptOpts = append(cryptOpts, fsDevice, "-")

		log.Infoln("Creating encrypted device")
		log.Debugln("Encrypting with command line: cryptsetup", strings.Join(cryptOpts, " "))
		if err := executil.CheckExecWithInput(inputQuery.EncryptionKey,"cryptsetup", cryptOpts...); err != nil {
			return errwrap.Wrap(errCryptSetupFailed, err)
		}

		log.Infoln("Opening encrypted device for filesystem setup")
		luksCtx, err := volumeaccess.OpenEncryptedDevice(inputQuery.EncryptionKey, fsDevice)
		if err != nil {
			return err
		}
		// Ensure the encrypted device will be unmounted when we finish here.
		defer func() {
			log.Infoln("Closing the encrypted device")
			if err := luksCtx.Close(); err != nil {
				log.Errorln("Error closing the encrypted device. Context may leak.")
			}
		}()
		// TODO: does this *ever* change across linux distros? How do you detect if it does?
		fsDevice = luksCtx.GetDevicePath()
		log.Debugln("fsDevice updated to cryptvolume is", fsDevice)
	} else {
		log.Debugln("fsDevice is data volume", fsDevice)
	}

	log.Infoln("Creating filesystem on device:", fsDevice)
	// TODO: there's some operator option we'd like to have here to default fs params
	// i.e. a pre-config which switches on filesystem type to change boot params
	filesystem := inputQuery.Filesystem

	mkfsOpts := []string{"-V", "-t", filesystem, fsDevice}
	log.Debugln("Creating filesystem with commandline: mkfs", strings.Join(mkfsOpts, " "))
	if err := executil.CheckExec("mkfs", mkfsOpts...); err != nil {
		return errwrap.Wrap(errFilesystemCreationFailed, err)
	}

	log.Infoln("Device initialization complete.")
	return nil
}