package probe

import (
	"os"
	"path/filepath"
	"testing"
)

func writeBattery(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, v := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(v+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// Batterie qui ne donne que charge_* (µAh) : la santé ne doit pas tomber à 0
func TestReadBatteryChargeOnly(t *testing.T) {
	dir := writeBattery(t, map[string]string{
		"type": "Battery", "capacity": "80", "status": "Discharging", "cycle_count": "0",
		"charge_full": "3000000", "charge_full_design": "4000000", "voltage_min_design": "11400000",
	})
	b, err := ReadBattery(dir)
	if err != nil {
		t.Fatal(err)
	}
	if b.DesignCapacity == 0 || b.CurrentCapacity == 0 {
		t.Fatalf("capacités non lues : %+v", b)
	}
	if h := b.CurrentCapacity / b.DesignCapacity; h < 0.749 || h > 0.751 {
		t.Fatalf("santé %.3f, attendu 0,75", h)
	}
	if b.CycleKnown {
		t.Fatal("cycle_count à 0 : nombre de cycles non communiqué")
	}
	if b.EnergyAH != 3000 {
		t.Fatalf("mAh = %v, attendu 3000", b.EnergyAH)
	}
}

func TestReadBatteryEnergy(t *testing.T) {
	dir := writeBattery(t, map[string]string{
		"type": "Battery", "capacity": "100", "status": "Full", "cycle_count": "312",
		"energy_full": "45000000", "energy_full_design": "50000000",
	})
	b, err := ReadBattery(dir)
	if err != nil {
		t.Fatal(err)
	}
	if b.CurrentCapacity != 45e6 || b.DesignCapacity != 50e6 || !b.CycleKnown || b.Cycle != 312 {
		t.Fatalf("lecture fausse : %+v", b)
	}
}

func TestIsSystemBattery(t *testing.T) {
	if !IsSystemBattery(writeBattery(t, map[string]string{"type": "Battery"})) {
		t.Fatal("batterie du PC non reconnue")
	}
	if IsSystemBattery(writeBattery(t, map[string]string{"type": "Battery", "scope": "Device"})) {
		t.Fatal("pile de souris prise pour la batterie du PC")
	}
	if IsSystemBattery(writeBattery(t, map[string]string{"type": "Mains"})) {
		t.Fatal("chargeur pris pour une batterie")
	}
}

func TestIsInternalDisk(t *testing.T) {
	root := t.TempDir()
	mk := func(rel, removable string) string {
		d := filepath.Join(root, rel)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		_ = os.WriteFile(filepath.Join(d, "removable"), []byte(removable+"\n"), 0o644)
		return d
	}
	if !IsInternalDisk(mk("pci0000:00/0000:00:17.0/ata1/sda", "0")) {
		t.Fatal("disque SATA exclu")
	}
	if IsInternalDisk(mk("pci0000:00/0000:00:14.0/usb2/2-1/sdb", "0")) {
		t.Fatal("clé USB comptée comme disque interne")
	}
	if IsInternalDisk(mk("virtual/sdc", "1")) {
		t.Fatal("disque amovible compté comme disque interne")
	}
	if IsInternalDisk(mk("virtual/loop0", "0")) {
		t.Fatal("disque virtuel compté")
	}
}
