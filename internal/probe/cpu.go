package probe

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// CPUInfo contient les informations du CPU
type CPUInfo struct {
	Architect  string
	VendorID   string
	ModelName  string
	NumberCore int
	CacheSize  int64
	FreqMaxMHz float64
	FreqMinMHz float64
}

const (
	vendorIDKey   = "vendor_id"
	modelNameKey  = "model name"
	numCoresKey   = "cpu cores"
	pathCpuInfo   = "/proc/cpuinfo"
	pathSystemCpu = "/sys/devices/system/cpu/"
)

// CacheID identifie un cache physique unique
type CacheID struct {
	Level      int
	Type       string
	SharedCPUs string
}

// readFile lit un fichier et retourne son contenu nettoyé
func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// readLevel lit le niveau de cache
func readLevel(index string) (int, error) {
	content, err := readFile(filepath.Join(index, "level"))
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(content)
}

// readType lit le type de cache
func readType(index string) (string, error) {
	return readFile(filepath.Join(index, "type"))
}

// readSharedCPU lit la liste des CPUs partageant ce cache
func readSharedCPU(index string) (string, error) {
	return readFile(filepath.Join(index, "shared_cpu_list"))
}

// readSize lit et parse la taille du cache
func readSize(index string) (int64, error) {
	sizeStr, err := readFile(filepath.Join(index, "size"))
	if err != nil {
		return 0, err
	}
	return parseSize(sizeStr)
}

// parseSize convertit "64K" → 65536 octets
func parseSize(sizeStr string) (int64, error) {
	if sizeStr == "" {
		return 0, fmt.Errorf("taille vide")
	}

	lastChar := sizeStr[len(sizeStr)-1]

	// Cas sans unité
	if lastChar >= '0' && lastChar <= '9' {
		return strconv.ParseInt(sizeStr, 10, 64)
	}

	value, err := strconv.ParseInt(sizeStr[:len(sizeStr)-1], 10, 64)
	if err != nil {
		return 0, err
	}

	switch lastChar {
	case 'K', 'k':
		return value << 10, nil // *1024
	case 'M', 'm':
		return value << 20, nil // *1024*1024
	case 'G', 'g':
		return value << 30, nil // *1024*1024*1024
	default:
		return 0, fmt.Errorf("unité inconnue: %c", lastChar)
	}
}

// getTotalCacheSize calcule la taille totale des caches avec déduplication
func getTotalCacheSize() (int64, error) {
	var totalCache int64
	seenCaches := make(map[CacheID]struct{}) // struct{} au lieu de bool

	cpuPaths, err := filepath.Glob(filepath.Join(pathSystemCpu, "cpu[0-9]*"))
	if err != nil {
		return 0, fmt.Errorf("glob CPUs: %w", err)
	}

	for _, cpu := range cpuPaths {
		indexPaths, err := filepath.Glob(filepath.Join(cpu, "cache", "index*"))
		if err != nil {
			continue
		}

		for _, index := range indexPaths {
			level, err := readLevel(index)
			if err != nil {
				continue
			}

			cacheType, err := readType(index)
			if err != nil {
				continue
			}

			sharedCPU, err := readSharedCPU(index)
			if err != nil {
				continue
			}

			cacheID := CacheID{
				Level:      level,
				Type:       cacheType,
				SharedCPUs: sharedCPU,
			}

			if _, seen := seenCaches[cacheID]; seen {
				continue
			}

			size, err := readSize(index)
			if err != nil {
				continue
			}

			totalCache += size
			seenCaches[cacheID] = struct{}{}
		}
	}

	return totalCache, nil
}

// parseCPUInfo parse les lignes de /proc/cpuinfo
func parseCPUInfo(lines []string) CPUInfo {
	info := CPUInfo{
		Architect: runtime.GOARCH, // Pas besoin de fonction séparée
	}

	for _, line := range lines {
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch key {
		case vendorIDKey:
			if info.VendorID == "" {
				info.VendorID = value
			}
		case modelNameKey:
			if info.ModelName == "" {
				info.ModelName = value
			}
		case numCoresKey:
			if cores, err := strconv.Atoi(value); err == nil {
				info.NumberCore = cores
			}
		}
	}

	return info
}

// readCPUInfo lit et filtre /proc/cpuinfo avec lecture bufférisée
func readCPUInfo() ([]string, error) {
	file, err := os.Open(pathCpuInfo)
	if err != nil {
		return nil, fmt.Errorf("ouverture %s: %w", pathCpuInfo, err)
	}
	defer file.Close()

	result := make([]string, 0, 32)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		// Optimisation : vérifier le préfixe avant de faire strings.HasPrefix complet
		if len(line) > 0 && (line[0] == 'm' || line[0] == 'c' || line[0] == 'v') {
			if strings.HasPrefix(line, modelNameKey) ||
				strings.HasPrefix(line, numCoresKey) ||
				strings.HasPrefix(line, vendorIDKey) {
				result = append(result, line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("lecture %s: %w", pathCpuInfo, err)
	}

	return result, nil
}

// readFreqKHz lit une fréquence en KHz depuis sysfs
func readFreqKHz(path string) (int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	freqStr := strings.TrimSpace(string(data))
	return strconv.ParseInt(freqStr, 10, 64)
}

// getCPUFrequencies retourne les fréquences min et max du CPU
func getCPUFrequencies() (minMHz, maxMHz float64, err error) {
	basePath := filepath.Join(pathSystemCpu, "cpu0", "cpufreq")

	maxKHz, err := readFreqKHz(filepath.Join(basePath, "cpuinfo_max_freq"))
	if err != nil {
		return 0, 0, fmt.Errorf("freq max: %w", err)
	}

	minKHz, err := readFreqKHz(filepath.Join(basePath, "cpuinfo_min_freq"))
	if err != nil {
		return 0, 0, fmt.Errorf("freq min: %w", err)
	}

	// Optimisation : division unique au lieu de deux constantes .0
	const khzToMhz = 1000.0
	return float64(minKHz) / khzToMhz, float64(maxKHz) / khzToMhz, nil
}

// GetCPUInfo retourne les informations complètes du CPU
func GetCPUInfo() (CPUInfo, error) {
	lines, err := readCPUInfo()
	if err != nil {
		return CPUInfo{}, err
	}

	info := parseCPUInfo(lines)

	cacheSize, err := getTotalCacheSize()
	if err != nil {
		return CPUInfo{}, fmt.Errorf("calcul cache: %w", err)
	}
	info.CacheSize = cacheSize

	// Fréquences optionnelles - ne pas échouer si absentes
	minMHz, maxMHz, _ := getCPUFrequencies()
	info.FreqMinMHz = minMHz
	info.FreqMaxMHz = maxMHz

	return info, nil
}
