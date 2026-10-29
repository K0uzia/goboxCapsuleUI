package probe

import (
	"os/exec"
	"strconv"
	"strings"
)

type MemoryInfo struct {
	Slots      []MemorySlot // barrettes installées
	EmptySlots []string     // emplacements sans barrette (Locator)
	SlotCount  int          // emplacements de la carte mère (« Number Of Devices »)
	TotalMB    int
}

type MemorySlot struct {
	Slot              string  // ex: Controller0-ChannelA-DIMM0
	BankLocator       string  // ex: BANK 0
	FormFactor        string  // ex: SODIMM, DIMM
	Type              string  // ex: DDR5
	TypeDetail        string  // ex: Synchronous
	Manufacturer      string  // ex: Samsung
	SerialNumber      string  // ex: 4735802A
	PartNumber        string  // ex: M425R2GA3BB0-CWMOD
	AssetTag          string  // ex: 9876543210
	SizeMB            int     // en MB
	Speed             int     // ex: 5600
	ConfiguredSpeed   int     // ex: 4800
	ConfiguredVoltage float64 // ex: 1.1
	Rank              int     // ex: 1
	Technology        string  // ex: DRAM
	OperatingMode     string  // ex: Volatile memory
}

func cutPrefixString(line, prefix string, field *string) {
	if after, ok := strings.CutPrefix(line, prefix); ok {
		*field = strings.TrimSpace(after)
	}
}

func parseIntField(line, prefix string, field *int, mult int, suffixes ...string) bool {
	if after, ok := strings.CutPrefix(line, prefix); ok {
		val := strings.TrimSpace(after)
		for _, s := range suffixes {
			val = strings.TrimSuffix(val, s)
		}
		val = strings.TrimSpace(val)

		if val == "" || val == "Unknown" || val == "0" || strings.Contains(val, "No Module") {
			return false
		}

		num, err := strconv.Atoi(val)
		if err == nil {
			*field = num * mult
			return true
		}
	}
	return false
}

func GetMemoryInfo() (MemoryInfo, error) {
	out, err := exec.Command("dmidecode", "-t", "memory").Output()
	if err != nil {
		return MemoryInfo{}, err
	}
	return parseMemory(out), nil
}

// parseSizeMB taille d'une barrette en Mo : « 16 GiB » (dmidecode ≥ 3.5),
// « 8192 MB » (versions plus anciennes) ; 0 si pas de barrette
func parseSizeMB(v string) int {
	f := strings.Fields(v)
	if len(f) != 2 {
		return 0
	}
	n, err := strconv.Atoi(f[0])
	if err != nil {
		return 0
	}
	switch f[1] {
	case "TB", "TiB":
		return n * 1024 * 1024
	case "GB", "GiB":
		return n * 1024
	case "MB", "MiB":
		return n
	case "kB", "KB", "KiB":
		return n / 1024
	}
	return 0
}

func parseMemory(data []byte) MemoryInfo {
	var info MemoryInfo
	seen := map[string]bool{}
	var slot MemorySlot
	inDevice, empty := false, false

	flush := func() {
		if !inDevice || seen[slot.Slot] {
			return
		}
		seen[slot.Slot] = true
		if slot.SizeMB > 0 {
			info.Slots = append(info.Slots, slot)
			info.TotalMB += slot.SizeMB
		} else if empty {
			info.EmptySlots = append(info.EmptySlots, slot.Slot)
		}
	}

	for line := range strings.SplitSeq(string(data)+"\n", "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "Memory Device":
			slot, inDevice, empty = MemorySlot{}, true, false
			continue
		case line == "":
			flush()
			inDevice = false
			continue
		}
		if v, ok := strings.CutPrefix(line, "Number Of Devices:"); ok {
			n, _ := strconv.Atoi(strings.TrimSpace(v))
			info.SlotCount += n
		}
		if !inDevice {
			continue
		}
		if v, ok := strings.CutPrefix(line, "Size:"); ok {
			v = strings.TrimSpace(v)
			slot.SizeMB = parseSizeMB(v)
			empty = strings.Contains(v, "No Module")
		}
		cutPrefixString(line, "Locator:", &slot.Slot)
		cutPrefixString(line, "Type:", &slot.Type)
		cutPrefixString(line, "Manufacturer:", &slot.Manufacturer)
		cutPrefixString(line, "Serial Number:", &slot.SerialNumber)
		cutPrefixString(line, "Bank Locator:", &slot.BankLocator)
		cutPrefixString(line, "Form Factor:", &slot.FormFactor)
		cutPrefixString(line, "Type Detail:", &slot.TypeDetail)
		cutPrefixString(line, "Part Number:", &slot.PartNumber)
		cutPrefixString(line, "Asset Tag:", &slot.AssetTag)
		cutPrefixString(line, "Memory Technology:", &slot.Technology)
		cutPrefixString(line, "Memory Operating Mode Capability:", &slot.OperatingMode)
		parseIntField(line, "Speed:", &slot.Speed, 1, "MT/s", "MHz")
		parseIntField(line, "Configured Memory Speed:", &slot.ConfiguredSpeed, 1, "MT/s", "MHz")
		parseIntField(line, "Rank:", &slot.Rank, 1)
	}
	return info
}
