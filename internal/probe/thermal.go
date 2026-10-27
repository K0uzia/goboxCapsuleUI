package probe

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gobox/internal/sysfs"
)

const (
	pathHwmon        = "/sys/class/hwmon"
	pathThermalZones = "/sys/class/thermal"
	pathProcStat     = "/proc/stat"
)

// Capteurs CPU connus, par ordre de préférence
var cpuHwmonNames = []string{"coretemp", "k10temp", "zenpower", "cpu_thermal", "acpitz"}

// CPUTemperature retourne la température du processeur en °C
// (capteur "Package" Intel, "Tctl" AMD, sinon le premier capteur trouvé).
func CPUTemperature() (float64, error) {
	entries, err := os.ReadDir(pathHwmon)
	if err == nil {
		byName := map[string]string{}
		for _, e := range entries {
			dir := filepath.Join(pathHwmon, e.Name())
			name, err := sysfs.ReadFile(filepath.Join(dir, "name"))
			if err == nil {
				byName[name] = dir
			}
		}
		for _, n := range cpuHwmonNames {
			if dir, ok := byName[n]; ok {
				if t, err := readHwmonTemp(dir); err == nil {
					return t, nil
				}
			}
		}
	}

	// Repli : zone thermique x86_pkg_temp ou première zone
	zones, _ := filepath.Glob(filepath.Join(pathThermalZones, "thermal_zone*"))
	var fallback string
	for _, z := range zones {
		typ, _ := sysfs.ReadFile(filepath.Join(z, "type"))
		if typ == "x86_pkg_temp" {
			fallback = z
			break
		}
		if fallback == "" {
			fallback = z
		}
	}
	if fallback != "" {
		if v, err := sysfs.ReadFloat(filepath.Join(fallback, "temp")); err == nil {
			return v / 1000, nil
		}
	}
	return 0, errors.New("aucun capteur de température CPU trouvé")
}

// readHwmonTemp lit le capteur "Package"/"Tctl" s'il existe, sinon temp1
func readHwmonTemp(dir string) (float64, error) {
	labels, _ := filepath.Glob(filepath.Join(dir, "temp*_label"))
	for _, l := range labels {
		label, _ := sysfs.ReadFile(l)
		if strings.HasPrefix(label, "Package") || label == "Tctl" || label == "Tdie" {
			input := strings.TrimSuffix(l, "_label") + "_input"
			if v, err := sysfs.ReadFloat(input); err == nil {
				return v / 1000, nil
			}
		}
	}
	v, err := sysfs.ReadFloat(filepath.Join(dir, "temp1_input"))
	if err != nil {
		return 0, err
	}
	return v / 1000, nil
}

// CPUFrequencyGHz retourne la fréquence moyenne actuelle des cœurs
func CPUFrequencyGHz() (float64, error) {
	files, _ := filepath.Glob(filepath.Join(pathSystemCpu, "cpu[0-9]*", "cpufreq", "scaling_cur_freq"))
	var sum float64
	var n int
	for _, f := range files {
		if v, err := sysfs.ReadFloat(f); err == nil {
			sum += v
			n++
		}
	}
	if n == 0 {
		return 0, errors.New("fréquence CPU indisponible")
	}
	return sum / float64(n) / 1e6, nil // kHz → GHz
}

// CPUTimes compteurs cumulés de /proc/stat, pour calculer la charge
type CPUTimes struct {
	Idle  uint64
	Total uint64
}

// ReadCPUTimes lit la ligne "cpu" agrégée de /proc/stat
func ReadCPUTimes() (CPUTimes, error) {
	data, err := os.ReadFile(pathProcStat)
	if err != nil {
		return CPUTimes{}, err
	}
	line, _, _ := strings.Cut(string(data), "\n")
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return CPUTimes{}, errors.New("format /proc/stat inattendu")
	}
	var t CPUTimes
	for i, f := range fields[1:] {
		v, _ := strconv.ParseUint(f, 10, 64)
		t.Total += v
		if i == 3 || i == 4 { // idle + iowait
			t.Idle += v
		}
	}
	return t, nil
}

// CPULoad retourne la charge (0–100 %) entre deux relevés
func CPULoad(prev, cur CPUTimes) float64 {
	total := cur.Total - prev.Total
	if total == 0 {
		return 0
	}
	return 100 * float64(total-(cur.Idle-prev.Idle)) / float64(total)
}
