package utils

import (
	"fmt"
	"path/filepath"

	"gobox/internal/sysfs"
)

func ConvertWattToAmpere(batteryPath string, energyFull float64) (float64, error) {
	voltagePath := filepath.Join(batteryPath, "voltage_min_design")

	voltDesign, err := sysfs.ReadFloat(voltagePath)
	if err != nil {
		return 0, fmt.Errorf("reading voltage: %w", err)
	}

	mAh := (energyFull / 1_000_000) * 1000 / (voltDesign / 1_000_000)
	return mAh, nil
}
