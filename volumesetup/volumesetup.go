package volumesetup

import (
	"os"
	"fmt"
	"errors"
	"strings"

	"github.com/hashicorp/errwrap"
	"github.com/wrouesnel/go.sysutil/executil"

	"github.com/wrouesnel/docker-simple-disk/volumequery"
	"github.com/wrouesnel/go.log"
	"github.com/satori/go.uuid"
)

var (
	errPartitioningFailed = errors.New("failed to partition disk")
	errPartProbeFailed = errors.New("informing kernel of partition update failed")
	errNoPartitionsFound = errors.New("no (0) disk partitions were found after disk partitioning")
	errPartitionNotFoundAfterPartitioning = errors.New("disk partition was not found after disk partitioning")
	errCouldNotWriteVolumeLabel = errors.New("failed to write volumelabel")
	errCryptSetupFailed = errors.New("error setting up encrypted volume")
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
		fmt.Sprintf("%d:%s",partIdx, volumequery.SimpleMetadataLabel),
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

	log.Infoln("Querying partitions from udev")
	partitions, err := volumequery.GetPartitionDevicesFromDiskPath(blockDevice)
	if err != nil {
		return errwrap.Wrap(errPartitionNotFoundAfterPartitioning, err)
	}
	// No partitions found!
	if len(partitions) == 0 {
		return errNoPartitionsFound
	}
	log.Debugln("Found partitions for device:", strings.Join(partitions, " "))

	// TODO: check that the partition is the volumelabel / query up the volume
	labelDevice := blockDevice + "p1"
	log.Debugln("Volume label device is:", labelDevice)

	log.Infoln("Writing volumelabel to device")
	labelBytes, err := volumequery.SerializeVolumeLabel(&label)
	if err != nil {
		return err
	}

	if err := WriteAndSyncExistingFile(labelDevice, labelBytes, os.FileMode(0600)); err != nil {
		return errwrap.Wrap(errCouldNotWriteVolumeLabel, err)
	}

	// Okay we got this far. We need to setup the data partition.

	// TODO: check the second partition exists / query it up
	var fsDevice string
	fsDevice = blockDevice + "p2"

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

		mountDevice := uuid.NewV4().String()
		cryptOpenOpts := []string{
			"-v",
			"open",
			fsDevice,
			mountDevice,
		}

		log.Infoln("Opening encrypted device for filesystem setup")
		log.Debugln("Opening encrypted device with command line: cryptsetup", strings.Join(cryptOpenOpts, " "))
		if err := executil.CheckExecWithInput(inputQuery.EncryptionKey, "cryptsetup", cryptOpenOpts...); err != nil {
			return errwrap.Wrap(errCryptSetupFailed, err)
		}

		// Ensure the encrypted device will be unmounted when we finish here.
		defer func() {
			log.Infoln("Closing the encrypted device")
			log.Debugln("Closing the encrypted device with command line: cryptsetup close", mountDevice)
			if err := executil.CheckExec("cryptsetup", "close", mountDevice); err != nil {
				log.Errorln("Error unmounting luksDevice:", err)
			}
		}()

		// TODO: does this *ever* change across linux distros?
		fsDevice = "/dev/mapper/" + mountDevice
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

