package probe

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	pathRoot        = "/sys/block/"
	sectorSizeBytes = 512
)

// DiskInfo contient les informations d'un disque
type DiskInfo struct {
	Name       string   // Nom du périphérique (ex: "sda", "nvme0n1")
	Vendor     string   // Fabricant (ex: "Samsung", peut être vide)
	Model      string   // Modèle (ex: "860 EVO", peut être vide)
	Type       string   // Type de disque : "SSD" ou "HDD"
	SizeBytes  int64    // Taille totale en octets
	Partitions []string // Liste des partitions (ex: ["nvme0n1p1", "nvme0n1p2"])
}

func GetDiskInfo(diskName string) (*DiskInfo, error) {
	if err := validateDiskName(diskName); err != nil {
		return nil, err
	}
	info := &DiskInfo{
		Name: diskName,
	}
	size, err := readDiskSize(diskName)
	if err != nil {
		return nil, fmt.Errorf("erreur lecture taille disque %s: %w", diskName, err)
	}

	info.SizeBytes = size // 2. Lire taille

	diskType, err := readDiskType(diskName)
	if err != nil {
		return nil, fmt.Errorf("erreur lecture type disque %s: %w", diskName, err)
	}

	info.Type = diskType // 3. Lire type
	vendor, model, err := readModelAndVendor(diskName)
	if err != nil {
		log.Printf("Warning: impossible de lire vendor/model du disque %s : %v", diskName, err)
		vendor, model = "", ""
	}

	info.Vendor = vendor // 4. Lire vendor
	info.Model = model   // 5. Lire modèle

	// Ajouter les partitions
	partitions, err := listPartitions(diskName)
	if err != nil {
		log.Printf("Warning: impossible de lire partitions du disque %s : %v", diskName, err)
		partitions = []string{} // Liste vide si erreur
	}
	info.Partitions = partitions

	return info, nil
}

func ListDisks() ([]string, error) {
	entries, err := os.ReadDir(pathRoot)
	if err != nil {
		return nil, fmt.Errorf("lecture %s: %w", pathRoot, err)
	}

	diskNames := make([]string, 0, len(entries))

	for _, entry := range entries {
		name := entry.Name()

		// Filtrer périphériques virtuels en premier (plus rapide)
		if strings.HasPrefix(name, "loop") ||
			strings.HasPrefix(name, "zram") ||
			strings.HasPrefix(name, "ram") {
			continue
		}

		// Vérifier si c'est un symlink pointant vers un répertoire
		if entry.Type()&os.ModeSymlink != 0 {
			fullPath := filepath.Join(pathRoot, name)
			info, err := os.Stat(fullPath)
			if err != nil || !info.IsDir() {
				continue
			}
			diskNames = append(diskNames, name)
		} else if entry.IsDir() {
			// Cas où ce serait un répertoire direct (rare dans /sys/block/)
			diskNames = append(diskNames, name)
		}
	}

	return diskNames, nil
}

func listPartitions(diskName string) ([]string, error) {
	diskPath := filepath.Join(pathRoot, diskName)

	entries, err := os.ReadDir(diskPath)
	if err != nil {
		return nil, fmt.Errorf("lecture %s: %w", diskPath, err)
	}

	partitions := make([]string, 0, 4)

	for _, entry := range entries {
		// Garder seulement répertoires (partitions sont des sous-répertoires)
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Vérifier si c'est une partition du disque
		if suffix, found := strings.CutPrefix(name, diskName); found {
			if suffix != "" && (strings.HasPrefix(suffix, "p") ||
				(len(suffix) > 0 && suffix[0] >= '0' && suffix[0] <= '9')) {
				partitions = append(partitions, name)
			}
		}
	}

	return partitions, nil
}

func validateDiskName(diskName string) error {
	if strings.Contains(diskName, "..") || strings.Contains(diskName, "/") {
		return fmt.Errorf("nom de disque invalide : %s", diskName)
	}
	if diskName == "" {
		return fmt.Errorf("nom de disque vide")
	}
	return nil
}

func readDiskSize(diskName string) (int64, error) {
	path := filepath.Join(pathRoot, diskName, "size")
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("lecture %s: %w", path, err)
	}

	dataStr := strings.TrimSpace(string(data))
	dataInt, err := strconv.ParseInt(dataStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("conversion taille disque %s: %w", diskName, err)
	}

	sizeBytes := dataInt * sectorSizeBytes
	return sizeBytes, nil
}

func readDiskType(diskName string) (string, error) {
	path := filepath.Join(pathRoot, diskName, "queue", "rotational")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("lecteur %s : %w", path, err)
	}
	dataStr := strings.TrimSpace(string(data))
	typeDIsk := "HDD"
	if dataStr == "0" {
		typeDIsk = "SSD"
	}
	return typeDIsk, nil
}

func readModelAndVendor(diskName string) (string, string, error) {
	// Chemins potentiels
	vendorPath := filepath.Join(pathRoot, diskName, "device", "vendor")
	modelPath := filepath.Join(pathRoot, diskName, "device", "model")

	// Lecture vendor (optionnel)
	vendor := ""
	if data, err := os.ReadFile(vendorPath); err == nil {
		vendor = strings.TrimSpace(string(data))
	} else if !os.IsNotExist(err) {
		// Si erreur autre que fichier absent, renvoyer erreur
		return "", "", fmt.Errorf("lecture vendor %s: %w", vendorPath, err)
	}

	// Lecture modèle (obligatoire)
	modelData, err := os.ReadFile(modelPath)
	if err != nil {
		return "", "", fmt.Errorf("lecture model %s: %w", modelPath, err)
	}
	modelStr := strings.TrimSpace(string(modelData))

	// Si vendor vide, tenter d'extraire vendor + modèle à partir de modelStr
	if vendor == "" {
		parts := strings.SplitN(modelStr, "_", 2)
		if len(parts) == 2 {
			vendor, modelStr = parts[0], parts[1]
		} else {
			// Pas de séparation possible, vendor vide, modèle complet
			vendor = ""
		}
	}

	return vendor, modelStr, nil
}
