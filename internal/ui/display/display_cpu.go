package ui

import (
	"fmt"

	"gobox/internal/probe"
)

func DisplayCPUInfo() {
	info, err := probe.GetCPUInfo()
	if err != nil {
		fmt.Printf("Erreur: %v\n", err)
		return
	}

	fmt.Printf("Vendor ID:       %s\n", info.VendorID)
	fmt.Printf("Model Name:      %s\n", info.ModelName)
	fmt.Printf("Architecture:    %s\n", info.Architect)
	fmt.Printf("CPU Cores:       %d\n", info.NumberCore)
	fmt.Printf("Total Cache:     %d octets (%.2f MB)\n",
		info.CacheSize, float64(info.CacheSize)/(1024*1024))

	if info.FreqMaxMHz > 0 {
		fmt.Printf("Freq Min:        %.0f MHz\n", info.FreqMinMHz)
		fmt.Printf("Freq Max:        %.0f MHz\n", info.FreqMaxMHz)
	}
}
