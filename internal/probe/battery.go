package probe

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gobox/internal/sysfs"
	"gobox/internal/utils"
)

type BatteryInfo struct {
	Status          string
	Manufacturer    *string
	Model           *string
	Serial          *string
	Technology      *string
	Capacity        int
	Cycle           int
	EnergyAH        float64
	VoltageNow      float64
	DesignCapacity  float64 // 🆕 Capacité théorique (neuve)
	CurrentCapacity float64 // 🆕 Capacité actuelle
}

var (
	batteryPath string
	once        sync.Once
)

const (
	pathPowerSupply = "/sys/class/power_supply"
)

func findBatteryPathCached() (string, error) {
	var err error
	once.Do(func() {
		batteryPath, err = findBatteryPath()
	})
	if batteryPath == "" || err != nil {
		return "", errors.New("battery path not found")
	}
	return batteryPath, nil
}

func strPtrIfNotEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func GetBatteryInfo() (BatteryInfo, error) {
	path, err := findBatteryPathCached()
	if err != nil {
		return BatteryInfo{}, err
	}

	// Required fields
	capacity, err := sysfs.ReadInt(filepath.Join(path, "capacity"))
	if err != nil {
		return BatteryInfo{}, fmt.Errorf("reading capacity: %w", err)
	}

	status, err := sysfs.ReadFile(filepath.Join(path, "status"))
	if err != nil {
		return BatteryInfo{}, fmt.Errorf("reading status: %w", err)
	}

	// Optional fields (no error handling, use empty string if missing)
	manufacturer, _ := sysfs.ReadFileOptional(filepath.Join(path, "manufacturer"))
	model, _ := sysfs.ReadFileOptional(filepath.Join(path, "model_name"))
	serialNumber, _ := sysfs.ReadFileOptional(filepath.Join(path, "serial_number"))
	technology, _ := sysfs.ReadFileOptional(filepath.Join(path, "technology"))

	cycle := 0
	if c, err := sysfs.ReadInt(filepath.Join(path, "cycle_count")); err == nil {
		cycle = c
	}

	voltNow := 0.0
	if v, err := sysfs.ReadFloat(filepath.Join(path, "voltage_now")); err == nil {
		voltNow = v
	}

	currentCapacity := 0.0
	if e, err := sysfs.ReadFloat(filepath.Join(path, "energy_full")); err == nil {
		currentCapacity = e
	}

	// Capacité théorique (neuve)
	designCapacity := 0.0
	if e, err := sysfs.ReadFloat(filepath.Join(path, "energy_full_design")); err == nil {
		designCapacity = e
	}

	// Convert Wh to Ah (silent error handling)
	energyAH, _ := utils.ConvertWattToAmpere(path, currentCapacity)

	return BatteryInfo{
		Status:          status,
		Manufacturer:    strPtrIfNotEmpty(manufacturer),
		Model:           strPtrIfNotEmpty(model),
		Serial:          strPtrIfNotEmpty(serialNumber),
		Technology:      strPtrIfNotEmpty(technology),
		Capacity:        capacity,
		Cycle:           cycle,
		EnergyAH:        energyAH,
		VoltageNow:      voltNow,
		CurrentCapacity: currentCapacity,
		DesignCapacity:  designCapacity,
	}, nil
}

func findBatteryPath() (string, error) {
	entries, err := os.ReadDir(pathPowerSupply)
	if err != nil {
		return "", fmt.Errorf("listing power_supply: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "BAT") {
			return filepath.Join(pathPowerSupply, name), nil
		}
	}
	return "", errors.New("no battery found")
}
