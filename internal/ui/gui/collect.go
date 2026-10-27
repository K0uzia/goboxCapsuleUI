package gui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	diagBattery "gobox/internal/diagnostic/battery"
	"gobox/internal/diagnostic/common"
	"gobox/internal/probe"
)

// Field une ligne "libellé : valeur" de la fiche technique
type Field struct {
	Label string
	Value string
}

// Component résultat de la détection d'un composant
type Component struct {
	Name    string
	Summary string       // résumé affiché sur le dashboard
	Fields  []Field      // détails affichés dans la fiche technique
	Grade   common.Grade // "" = pas encore de notation pour ce composant
	Err     error
}

// collectInventory détecte tous les composants (lent : dmidecode, sysfs…)
func collectInventory() []Component {
	return []Component{
		collectCPU(), collectRAM(), collectDisks(), collectGPU(), collectNetwork(), collectBattery(),
	}
}

// machineInfo identité de la machine (DMI)
type machineInfo struct {
	Vendor, Product, Serial string
}

func (m machineInfo) Name() string {
	name := strings.TrimSpace(m.Vendor + " " + m.Product)
	if name == "" {
		name, _ = os.Hostname()
	}
	return name
}

// readMachine lit /sys/class/dmi/id (le n° de série nécessite root)
func readMachine() machineInfo {
	read := func(f string) string {
		b, _ := os.ReadFile("/sys/class/dmi/id/" + f)
		v := strings.TrimSpace(string(b))
		if v == "To Be Filled By O.E.M." || v == "Default string" {
			return ""
		}
		return v
	}
	return machineInfo{Vendor: read("sys_vendor"), Product: read("product_name"), Serial: read("product_serial")}
}

func collectCPU() Component {
	c := Component{Name: "Processeur"}
	info, err := probe.GetCPUInfo()
	if err != nil {
		c.Err = err
		return c
	}
	c.Summary = info.ModelName
	c.Fields = []Field{
		{"Modèle", info.ModelName},
		{"Fabricant", info.VendorID},
		{"Architecture", info.Architect},
		{"Cœurs", strconv.Itoa(info.NumberCore)},
		{"Cache total", formatBytes(info.CacheSize)},
	}
	if info.FreqMaxMHz > 0 {
		c.Fields = append(c.Fields,
			Field{"Fréquence", fmt.Sprintf("%.0f – %.0f MHz", info.FreqMinMHz, info.FreqMaxMHz)})
	}
	return c
}

func collectRAM() Component {
	c := Component{Name: "Mémoire"}
	info, err := probe.GetMemoryInfo()
	if err != nil || len(info.Slots) == 0 {
		// dmidecode nécessite root : on se rabat sur /proc/meminfo
		totalKB, merr := readMemTotalKB()
		if merr != nil {
			c.Err = merr
			return c
		}
		c.Summary = fmt.Sprintf("%.1f Go", float64(totalKB)/(1024*1024))
		c.Fields = []Field{
			{"Total", c.Summary},
			{"Barrettes", "détails indisponibles (lancer en root)"},
		}
		return c
	}

	c.Summary = fmt.Sprintf("%.0f Go %s", float64(info.TotalMB)/1024, info.Slots[0].Type)
	c.Fields = []Field{{"Total", fmt.Sprintf("%.1f Go", float64(info.TotalMB)/1024)}}
	for _, s := range info.Slots {
		c.Fields = append(c.Fields, Field{
			s.Slot,
			fmt.Sprintf("%d Mo %s %s @ %d MHz · %s", s.SizeMB, s.Type, s.FormFactor, s.Speed, s.Manufacturer),
		})
	}
	return c
}

func collectDisks() Component {
	c := Component{Name: "Stockage"}
	names, err := probe.ListDisks()
	if err != nil {
		c.Err = err
		return c
	}
	if len(names) == 0 {
		c.Summary = "Aucun disque détecté"
		return c
	}

	var total int64
	for _, name := range names {
		d, err := probe.GetDiskInfo(name)
		if err != nil {
			c.Fields = append(c.Fields, Field{name, "erreur : " + err.Error()})
			continue
		}
		total += d.SizeBytes
		label := strings.TrimSpace(d.Vendor + " " + d.Model)
		if label == "" {
			label = "modèle inconnu"
		}
		c.Fields = append(c.Fields, Field{
			d.Name,
			fmt.Sprintf("%s · %s · %s · %d partition(s)", label, d.Type, formatBytes(d.SizeBytes), len(d.Partitions)),
		})
	}
	c.Summary = fmt.Sprintf("%d disque(s) · %s", len(names), formatBytes(total))
	return c
}

func collectGPU() Component {
	c := Component{Name: "Carte graphique"}
	gpus, err := probe.DetectGPUs()
	if err != nil {
		c.Err = err
		return c
	}
	if len(gpus) == 0 {
		c.Summary = "Aucune carte détectée"
		return c
	}
	c.Summary = gpus[0].Model
	for i, g := range gpus {
		prefix := fmt.Sprintf("GPU %d", i+1)
		c.Fields = append(c.Fields,
			Field{prefix, g.Model},
			Field{"  Fabricant", g.Vendor},
			Field{"  Driver", strings.TrimSpace(g.Driver + " " + g.Version)},
			Field{"  Sorties", strings.Join(g.Outputs, ", ")},
		)
	}
	return c
}

func collectNetwork() Component {
	c := Component{Name: "Réseau"}
	ifaces, err := probe.ListNetworkInterfaces()
	if err != nil {
		c.Err = err
		return c
	}
	connected := 0
	for _, n := range ifaces {
		state := "déconnecté"
		if n.IsUp && n.Carrier {
			state = "connecté"
			connected++
		}
		value := fmt.Sprintf("%s · %s · %s", n.Type, state, n.MACAddress)
		if n.Speed != "" {
			value += " · " + n.Speed
		}
		if n.IPAddress != "" {
			value += " · " + n.IPAddress
		}
		c.Fields = append(c.Fields, Field{n.Name, value})
	}
	c.Summary = fmt.Sprintf("%d interface(s) · %d connectée(s)", len(ifaces), connected)
	return c
}

func collectBattery() Component {
	c := Component{Name: "Batterie"}
	info, err := probe.GetBatteryInfo()
	if err != nil || info.Capacity < 0 {
		c.Summary = "Pas de batterie détectée"
		return c
	}

	result, err := diagBattery.RunBatteryTest()
	if err != nil {
		c.Err = err
		return c
	}
	c.Grade = result.Grade
	c.Summary = fmt.Sprintf("Santé %.0f %% · %d cycles", result.HealthPercentage, result.CycleCount)
	c.Fields = []Field{
		{"Charge", fmt.Sprintf("%d %% (%s)", info.Capacity, info.Status)},
		{"Santé", fmt.Sprintf("%.1f %%", result.HealthPercentage)},
		{"Cycles", strconv.Itoa(result.CycleCount)},
		{"Capacité", fmt.Sprintf("%.0f mAh", info.EnergyAH)},
	}
	if info.Manufacturer != nil {
		c.Fields = append(c.Fields, Field{"Fabricant", *info.Manufacturer})
	}
	if info.Model != nil {
		c.Fields = append(c.Fields, Field{"Modèle", *info.Model})
	}
	for _, issue := range result.Issues {
		c.Fields = append(c.Fields, Field{"⚠ Problème", issue})
	}
	return c
}

func readMemTotalKB() (int64, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) >= 2 && fields[0] == "MemTotal:" {
			return strconv.ParseInt(fields[1], 10, 64)
		}
	}
	return 0, fmt.Errorf("MemTotal introuvable dans /proc/meminfo")
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d o", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %co", float64(b)/float64(div), "KMGTPE"[exp])
}
