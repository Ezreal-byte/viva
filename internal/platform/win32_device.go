package platform

import "time"

// DeviceLostCode reports whether an HRESULT says the graphics device is gone
// and must be made again: removed (a driver reset, a hung GPU timed out, the
// adapter unplugged or switched), reset, hung, or failed inside the driver.
func DeviceLostCode(code uint32) bool {
	switch code {
	case 0x887A0005, // DXGI_ERROR_DEVICE_REMOVED
		0x887A0006, // DXGI_ERROR_DEVICE_HUNG
		0x887A0007, // DXGI_ERROR_DEVICE_RESET
		0x887A0020: // DXGI_ERROR_DRIVER_INTERNAL_ERROR
		return true
	}
	return false
}

// DeviceRecoveryPlan is what to do about a lost graphics device.
type DeviceRecoveryPlan uint8

const (
	// RecoverHardware makes the device again, on the GPU if it can.
	RecoverHardware DeviceRecoveryPlan = iota
	// RecoverSoftware makes it on the software rasterizer, which a GPU that keeps
	// failing cannot take down.
	RecoverSoftware
	// RecoverGiveUp stops: the device was lost again and again.
	RecoverGiveUp
)

// DeviceRecovery decides how many times in a row a window may recover from a
// lost device. Losses far enough apart are independent: a laptop that switches
// GPUs now and then recovers every time. Losses close together mean the GPU is
// failing: the first two retry the hardware, the third moves to the software
// rasterizer and the fourth gives up, so a broken driver does not spin.
type DeviceRecovery struct {
	losses [4]time.Time
	count  int
}

// DeviceRecoveryWindow is how close together losses count as repeated.
const DeviceRecoveryWindow = 30 * time.Second

// Next records a loss at now and returns what to do about it.
func (recovery *DeviceRecovery) Next(now time.Time) DeviceRecoveryPlan {
	kept := 0
	for index := 0; index < recovery.count; index++ {
		if now.Sub(recovery.losses[index]) < DeviceRecoveryWindow {
			recovery.losses[kept] = recovery.losses[index]
			kept++
		}
	}
	recovery.count = kept
	if recovery.count < len(recovery.losses) {
		recovery.losses[recovery.count] = now
		recovery.count++
	}
	switch {
	case recovery.count <= 2:
		return RecoverHardware
	case recovery.count == 3:
		return RecoverSoftware
	}
	return RecoverGiveUp
}
