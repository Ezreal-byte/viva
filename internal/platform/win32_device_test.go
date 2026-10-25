package platform

import (
	"testing"
	"time"
)

func TestOnlyDeviceFailuresAreRecoverable(t *testing.T) {
	for _, code := range []uint32{0x887A0005, 0x887A0006, 0x887A0007, 0x887A0020} {
		if !DeviceLostCode(code) {
			t.Errorf("%#x is a lost device", code)
		}
	}
	for _, code := range []uint32{0, 0x087A0001 /* DXGI_STATUS_OCCLUDED */, 0x8007000E /* E_OUTOFMEMORY */, 0x887A0001} {
		if DeviceLostCode(code) {
			t.Errorf("%#x is not a lost device", code)
		}
	}
}

func TestRepeatedLossesEscalateAndThenStop(t *testing.T) {
	var recovery DeviceRecovery
	start := time.Unix(1000, 0)
	want := []DeviceRecoveryPlan{RecoverHardware, RecoverHardware, RecoverSoftware, RecoverGiveUp, RecoverGiveUp}
	for index, plan := range want {
		if got := recovery.Next(start.Add(time.Duration(index) * time.Second)); got != plan {
			t.Fatalf("loss %d: plan %d, want %d", index+1, got, plan)
		}
	}
}

func TestLossesFarApartAreIndependent(t *testing.T) {
	var recovery DeviceRecovery
	now := time.Unix(1000, 0)
	for range 20 {
		if got := recovery.Next(now); got != RecoverHardware {
			t.Fatalf("an isolated loss planned %d", got)
		}
		now = now.Add(DeviceRecoveryWindow + time.Second)
	}
}

func TestOldLossesAgeOutOfARun(t *testing.T) {
	var recovery DeviceRecovery
	now := time.Unix(1000, 0)
	recovery.Next(now)
	recovery.Next(now.Add(time.Second))
	later := now.Add(DeviceRecoveryWindow + 2*time.Second)
	if got := recovery.Next(later); got != RecoverHardware {
		t.Fatalf("after the run aged out, plan %d", got)
	}
}
