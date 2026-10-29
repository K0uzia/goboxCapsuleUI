package probe

import "testing"

// dmidecode 3.6 (Rocky 10) écrit « 16 GiB », les versions plus anciennes
// « 8192 MB » ; les emplacements vides sont comptés à part.
func TestParseMemory(t *testing.T) {
	out := []byte(`Physical Memory Array
	Number Of Devices: 3

Memory Device
	Size: 16 GiB
	Form Factor: SODIMM
	Locator: DIMM1
	Bank Locator: Not Specified
	Type: DDR4
	Speed: 2400 MT/s
	Manufacturer: 80AD000080AD
	Configured Memory Speed: 2133 MT/s

Memory Device
	Size: 8192 MB
	Locator: DIMM2
	Type: DDR4

Memory Device
	Size: No Module Installed
	Locator: DIMM3
	Type: Unknown
`)
	info := parseMemory(out)
	if len(info.Slots) != 2 || info.TotalMB != 16384+8192 {
		t.Fatalf("barrettes = %d, total = %d Mo", len(info.Slots), info.TotalMB)
	}
	if s := info.Slots[0]; s.Slot != "DIMM1" || s.BankLocator != "Not Specified" || s.Speed != 2400 || s.ConfiguredSpeed != 2133 {
		t.Fatalf("première barrette mal lue : %+v", s)
	}
	if len(info.EmptySlots) != 1 || info.EmptySlots[0] != "DIMM3" || info.SlotCount != 3 {
		t.Fatalf("emplacements : vides %v, total %d", info.EmptySlots, info.SlotCount)
	}
}
