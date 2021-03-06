package volumeaccess

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/hashicorp/errwrap"
	"github.com/satori/go.uuid"
	"github.com/wrouesnel/go.log"
	"github.com/wrouesnel/go.sysutil/executil"
)

var (
	errCryptSetupOpenFailed  = errors.New("error invoking cryptsetup to open device")
	errCryptSetupCloseFailed = errors.New("error invoking cryptsetup to close device")
)

type VolumeContext interface {
	// Return the device path of the context where it can be accessed
	GetDevicePath() string
	// Tear down the volume setup
	Close() error
}

// deviceContext represents the context of an unencrypted device.
type deviceContext struct {
	sourceDevicePath string
}

func OpenDevice(devicePath string) (VolumeContext, error) {
	return VolumeContext(&deviceContext{
		sourceDevicePath: devicePath,
	}), nil
}

// GetDevicePath returns the unencrypted device path
func (this *deviceContext) GetDevicePath() string {
	return this.sourceDevicePath
}

func (this *deviceContext) Close() error {
	// Nothing to actually.
	return nil
}

// encryptedDeviceContext represents the context of an opened LUKS volume.
type encryptedDeviceContext struct {
	sourceDevicePath string
	mountId          string
}

// OpenEncryptedDevice opens a given device as an encrypted device and returns
// the mount path.
func OpenEncryptedDevice(key string, devicePath string) (VolumeContext, error) {
	mountDevice := uuid.NewV4().String()
	cryptOpenOpts := []string{
		"-v",
		"open",
		devicePath,
		mountDevice,
	}

	log.Debugln("Opening encrypted device with command line: cryptsetup", strings.Join(cryptOpenOpts, " "))
	if err := executil.CheckExecWithInput(key, "cryptsetup", cryptOpenOpts...); err != nil {
		return nil, errwrap.Wrap(errCryptSetupOpenFailed, err)
	}

	newCtx := VolumeContext(&encryptedDeviceContext{
		sourceDevicePath: devicePath,
		mountId:          mountDevice,
	})

	return newCtx, nil
}

// GetDevicePath returns the unencrypted device path
func (this *encryptedDeviceContext) GetDevicePath() string {
	realPath, err := filepath.EvalSymlinks("/dev/mapper/" + this.mountId)
	if err != nil {
		return ""
	}

	return realPath
}

func (this *encryptedDeviceContext) Close() error {
	if err := executil.CheckExec("cryptsetup", "close", this.mountId); err != nil {
		log.Errorln("Error unmounting luksDevice:", err)
		return errwrap.Wrap(errCryptSetupCloseFailed, err)
	}
	return nil
}
