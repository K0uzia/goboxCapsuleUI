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
	CycleKnown      bool // faux : le microprogramme ne compte pas les cycles
	EnergyAH        float64
	VoltageNow      float64
	DesignCapacity  float64 // capacité neuve, µWh (0 : non communiquée)
	CurrentCapacity float64 // capacité actuelle, µWh (0 : non communiquée)
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
	return ReadBattery(path)
}

// ReadBattery lit une batterie de /sys/class/power_supply. Selon le pilote, les
// capacités sont en énergie (energy_*, µWh) ou en charge (charge_*, µAh) : la
// charge est convertie en énergie avec la tension nominale, pour que santé et
// affichage aient la même unité.
func ReadBattery(path string) (BatteryInfo, error) {
	capacity, err := sysfs.ReadInt(filepath.Join(path, "capacity"))
	if err != nil {
		return BatteryInfo{}, fmt.Errorf("reading capacity: %w", err)
	}

	status, err := sysfs.ReadFile(filepath.Join(path, "status"))
	if err != nil {
		return BatteryInfo{}, fmt.Errorf("reading status: %w", err)
	}

	manufacturer, _ := sysfs.ReadFileOptional(filepath.Join(path, "manufacturer"))
	model, _ := sysfs.ReadFileOptional(filepath.Join(path, "model_name"))
	serialNumber, _ := sysfs.ReadFileOptional(filepath.Join(path, "serial_number"))
	technology, _ := sysfs.ReadFileOptional(filepath.Join(path, "technology"))

	num := func(f string) float64 {
		v, err := sysfs.ReadFloat(filepath.Join(path, f))
		if err != nil || v < 0 {
			return 0
		}
		return v
	}

	cycle, cycleErr := sysfs.ReadInt(filepath.Join(path, "cycle_count"))
	if cycleErr != nil || cycle < 0 {
		cycle = 0
	}

	voltNow := num("voltage_now")
	volt := num("voltage_min_design")
	if volt == 0 {
		volt = voltNow
	}
	energy := func(name string) float64 {
		if e := num("energy_" + name); e > 0 {
			return e
		}
		return num("charge_"+name) * volt / 1e6 // µAh × µV
	}
	currentCapacity, designCapacity := energy("full"), energy("full_design")

	energyAH := num("charge_full") / 1000 // µAh → mAh
	if energyAH == 0 {
		energyAH, _ = utils.ConvertWattToAmpere(path, currentCapacity)
	}

	return BatteryInfo{
		Status:          status,
		Manufacturer:    strPtrIfNotEmpty(manufacturer),
		Model:           strPtrIfNotEmpty(model),
		Serial:          strPtrIfNotEmpty(serialNumber),
		Technology:      strPtrIfNotEmpty(technology),
		Capacity:        capacity,
		Cycle:           cycle,
		CycleKnown:      cycleErr == nil && cycle > 0,
		EnergyAH:        energyAH,
		VoltageNow:      voltNow,
		CurrentCapacity: currentCapacity,
		DesignCapacity:  designCapacity,
	}, nil
}

// IsSystemBattery vrai pour la batterie du PC : type Battery, sans les piles d'une
// souris ou d'un clavier sans fil (scope Device). Le nom varie : BAT0, BAT1,
// CMB0, battery…
func IsSystemBattery(dir string) bool {
	name := strings.ToUpper(filepath.Base(dir))
	typ, err := sysfs.ReadFile(filepath.Join(dir, "type"))
	switch {
	case err != nil:
		// type illisible : on se fie au nom, comme avant
		return strings.HasPrefix(name, "BAT")
	case !strings.EqualFold(typ, "Battery"):
		return false
	}
	scope, _ := sysfs.ReadFileOptional(filepath.Join(dir, "scope"))
	return !strings.EqualFold(scope, "Device") && !strings.HasPrefix(name, "HID")
}

func findBatteryPath() (string, error) {
	entries, err := os.ReadDir(pathPowerSupply)
	if err != nil {
		return "", fmt.Errorf("listing power_supply: %w", err)
	}
	for _, entry := range entries {
		if dir := filepath.Join(pathPowerSupply, entry.Name()); IsSystemBattery(dir) {
			return dir, nil
		}
	}
	return "", errors.New("no battery found")
}
