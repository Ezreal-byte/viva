package linux

import (
	"sync"

	"github.com/ebitengine/purego"
)

// vkPhysicalDeviceTypeCPU is the type a software rasterizer reports (lavapipe).
const vkPhysicalDeviceTypeCPU = 4

var softwareProbe struct {
	once     sync.Once
	software bool
}

// vulkanDeviceIsSoftware reports whether the device a window would draw with is a
// CPU rasterizer: the first device with a graphics queue, the one the renderer
// picks. The answer decides whether the window gets a frame with a transparent
// margin (window_margin_linux.go): older software drivers cannot present to a
// window with alpha. It asks once; a machine with no Vulkan says it is not, and
// the renderer then fails with its own error.
func vulkanDeviceIsSoftware() bool {
	softwareProbe.once.Do(func() {
		api, err := loadVulkan()
		if err != nil {
			return
		}
		defer purego.Dlclose(api.library)
		renderer := &vulkanRenderer{api: api}
		if err := renderer.createInstance(nil); err != nil {
			return
		}
		defer api.destroyInstance(renderer.instance, nil)
		var properties func(uintptr, *[824]byte)
		purego.RegisterLibFunc(&properties, api.library, "vkGetPhysicalDeviceProperties")
		var count uint32
		if api.enumeratePhysicalDevices(renderer.instance, &count, nil) != vkSuccess || count == 0 || count > 32 {
			return
		}
		devices := make([]uintptr, count)
		if api.enumeratePhysicalDevices(renderer.instance, &count, &devices[0]) != vkSuccess {
			return
		}
		for _, device := range devices[:count] {
			var families uint32
			api.getQueueFamilyProperties(device, &families, nil)
			if families == 0 || families > 64 {
				continue
			}
			list := make([]vkQueueFamilyProperties, families)
			api.getQueueFamilyProperties(device, &families, &list[0])
			for _, family := range list[:families] {
				if family.Flags&vkQueueGraphics != 0 && family.Count > 0 {
					var buffer [824]byte
					properties(device, &buffer)
					kind := uint32(buffer[16]) | uint32(buffer[17])<<8 | uint32(buffer[18])<<16 | uint32(buffer[19])<<24
					softwareProbe.software = kind == vkPhysicalDeviceTypeCPU
					return
				}
			}
		}
	})
	return softwareProbe.software
}
