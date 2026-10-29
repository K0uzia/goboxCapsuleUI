package gui

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	Icon    string       // icône de la fiche (icons/)
	Summary string       // résumé affiché sur le dashboard
	Fields  []Field      // détails affichés dans la fiche technique
	Grade   common.Grade // "" = pas encore de notation pour ce composant
	Err     error
}

// collectInventory détecte tous les composants (lent : dmidecode, sysfs…),
// dans l'ordre de la fiche papier
func collectInventory() []Component {
	return []Component{
		collectCPU(), collectRAM(), collectDisks(), collectBattery(), collectNetwork(), collectGPU(), collectDisplay(),
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
	c := Component{Name: "Processeur", Icon: "microchip"}
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
			Field{"Fréquence", fmt.Sprintf("%.0f à %.0f MHz", info.FreqMinMHz, info.FreqMaxMHz)})
	}
	return c
}

func collectRAM() Component {
	c := Component{Name: "Mémoire RAM", Icon: "memory"}
	info, err := probe.GetMemoryInfo()
	if err != nil || len(info.Slots) == 0 {
		// dmidecode nécessite root : on se rabat sur /proc/meminfo
		totalKB, merr := readMemTotalKB()
		if merr != nil {
			c.Err = merr
			return c
		}
		why := "dmidecode n'a lu aucune barrette"
		switch {
		case !isAdmin():
			why = "droits administrateur nécessaires"
		case err != nil:
			why = "dmidecode : " + err.Error()
			var exit *exec.ExitError
			if errors.As(err, &exit) && len(exit.Stderr) > 0 {
				why += ", " + strings.TrimSpace(string(exit.Stderr))
			}
		}
		c.Summary = fmt.Sprintf("%.1f Go", float64(totalKB)/(1024*1024))
		c.Fields = []Field{
			{"Total", c.Summary},
			{"Barrettes", "détails indisponibles (" + why + ")"},
		}
		return c
	}

	c.Summary = fmt.Sprintf("%s %s", formatMB(info.TotalMB), info.Slots[0].Type)
	c.Fields = []Field{{"Total", formatMB(info.TotalMB)}}
	if info.SlotCount > 0 {
		c.Fields = append(c.Fields, Field{"Emplacements", fmt.Sprintf("%d, dont %d occupé(s)", info.SlotCount, len(info.Slots))})
	}
	for _, sl := range info.Slots {
		v := strings.TrimSpace(fmt.Sprintf("%s %s %s", formatMB(sl.SizeMB), sl.Type, sl.FormFactor))
		if sl.Speed > 0 {
			v += fmt.Sprintf(" · %d MT/s", sl.Speed)
			if sl.ConfiguredSpeed > 0 && sl.ConfiguredSpeed != sl.Speed {
				v += fmt.Sprintf(" (%d en service)", sl.ConfiguredSpeed)
			}
		}
		if m := strings.TrimSpace(memoryVendor(sl.Manufacturer) + " " + strings.TrimSpace(sl.PartNumber)); m != "" {
			v += " · " + m
		}
		c.Fields = append(c.Fields, Field{sl.Slot, v})
	}
	for _, e := range info.EmptySlots {
		c.Fields = append(c.Fields, Field{e, "vide"})
	}
	return c
}

func collectDisks() Component {
	c := Component{Name: "Stockage", Icon: "hard-drive"}
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
	c := Component{Name: "Carte graphique", Icon: "display"}
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
		label := "Modèle"
		if len(gpus) > 1 {
			label = fmt.Sprintf("Carte %d", i+1)
		}
		c.Fields = append(c.Fields,
			Field{label, g.Model},
			Field{"Fabricant", g.Vendor},
			Field{"Pilote", strings.TrimSpace(g.Driver + " " + strings.TrimSuffix(g.Version, "inconnu"))},
		)
	}
	return c
}

func collectNetwork() Component {
	c := Component{Name: "Cartes réseau", Icon: "wifi"}
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
		if chip := netChip(n.Name); chip != "" {
			value = chip + " · " + value
		}
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
	c := Component{Name: "Batterie", Icon: "battery-half"}
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
	c.Fields = []Field{{"Charge", fmt.Sprintf("%d %% (%s)", info.Capacity, batteryStatus(info.Status))}}
	b := readBatterySys()
	if b.remaining > 0 {
		c.Fields = append(c.Fields, Field{"Temps restant", formatClock(b.remaining)})
	}
	if info.DesignCapacity > 0 {
		c.Fields = append(c.Fields, Field{"Capacité d'origine", fmt.Sprintf("%.1f Wh", info.DesignCapacity/1e6)})
	}
	if info.CurrentCapacity > 0 {
		c.Fields = append(c.Fields, Field{"Capacité réelle", fmt.Sprintf("%.1f Wh, soit %.0f %%", info.CurrentCapacity/1e6, result.HealthPercentage)})
	}
	if info.EnergyAH > 0 {
		c.Fields = append(c.Fields, Field{"Capacité (mAh)", fmt.Sprintf("%.0f mAh", info.EnergyAH)})
	}
	c.Fields = append(c.Fields, Field{"Cycles", strconv.Itoa(result.CycleCount)})
	for _, f := range []struct {
		label string
		v     *string
	}{{"Fabricant", info.Manufacturer}, {"Modèle", info.Model}, {"Série", info.Serial}, {"Technologie", info.Technology}} {
		if f.v != nil {
			c.Fields = append(c.Fields, Field{f.label, *f.v})
		}
	}
	if b.power > 0 {
		c.Fields = append(c.Fields, Field{"Puissance", fmt.Sprintf("%.1f W", b.power)})
	}
	if info.VoltageNow > 0 {
		c.Fields = append(c.Fields, Field{"Tension", fmt.Sprintf("%.2f V", info.VoltageNow/1e6)})
	}
	for _, issue := range result.Issues {
		c.Fields = append(c.Fields, Field{"Problème", issue})
	}
	return c
}

// batteryStatus état de charge en français
func batteryStatus(s string) string {
	switch s {
	case "Charging":
		return "en charge"
	case "Discharging":
		return "sur batterie"
	case "Full":
		return "pleine"
	case "Not charging":
		return "branchée, sans charger"
	}
	return strings.ToLower(s)
}

// batterySys puissance instantanée et autonomie restante (sur batterie)
type batterySys struct {
	power     float64 // W
	remaining time.Duration
}

func readBatterySys() batterySys {
	var b batterySys
	dirs, _ := filepath.Glob("/sys/class/power_supply/BAT*")
	if len(dirs) == 0 {
		return b
	}
	d := dirs[0]
	num := func(f string) float64 {
		v, _ := strconv.ParseFloat(readTrim(filepath.Join(d, f)), 64)
		return v
	}
	power := num("power_now") // µW
	if power == 0 {
		power = num("current_now") * num("voltage_now") / 1e6 // µA × µV
	}
	b.power = power / 1e6
	energy := num("energy_now") // µWh
	if energy == 0 {
		energy = num("charge_now") * num("voltage_now") / 1e6
	}
	if readTrim(filepath.Join(d, "status")) == "Discharging" && power > 0 {
		b.remaining = time.Duration(energy / power * float64(time.Hour))
	}
	return b
}

// netChip nom de la puce d'une interface réseau (PCI ou USB)
func netChip(iface string) string {
	dev, err := filepath.EvalSymlinks(filepath.Join("/sys/class/net", iface, "device"))
	if err != nil {
		return ""
	}
	if strings.Contains(dev, "/usb") {
		return readTrim(filepath.Join(dev, "..", "product"))
	}
	return pciName(filepath.Base(dev))
}

// collectDisplay écran intégré et sorties vidéo, connectées ou non
func collectDisplay() Component {
	c := Component{Name: "Affichage", Icon: "tv"}
	conns := videoConnectors()
	if len(conns) == 0 {
		c.Summary = "Aucune sortie détectée"
		return c
	}
	connected := 0
	for _, v := range conns {
		state := "déconnecté"
		if v.Connected {
			state = strings.TrimSpace("connecté " + v.Mode)
			connected++
		}
		label := v.Type + " (" + v.Name + ")"
		if v.Type == internalScreen {
			label = internalScreen
		}
		c.Fields = append(c.Fields, Field{label, state})
	}
	c.Summary = fmt.Sprintf("%d écran(s) connecté(s)", connected)
	return c
}

// formatMB « 16 Go », « 512 Mo »
func formatMB(mb int) string {
	if mb >= 1024 && mb%1024 == 0 {
		return fmt.Sprintf("%d Go", mb/1024)
	}
	if mb >= 1024 {
		return fmt.Sprintf("%.1f Go", float64(mb)/1024)
	}
	return fmt.Sprintf("%d Mo", mb)
}

// jedecVendors codes fabricant JEDEC des barrettes les plus courantes (les 4
// premiers chiffres hexadécimaux, bit de parité compris)
var jedecVendors = map[string]string{
	"80AD": "SK Hynix", "00AD": "SK Hynix", "80CE": "Samsung", "00CE": "Samsung",
	"802C": "Micron", "002C": "Micron", "859B": "Crucial", "0198": "Kingston", "8198": "Kingston",
	"04CB": "A-DATA", "84CB": "A-DATA", "04CD": "G.Skill", "029E": "Corsair", "859E": "Corsair",
	"8551": "Qimonda", "80FE": "Elpida", "0443": "Ramaxel", "8304": "Transcend", "014F": "Transcend",
	"868A": "Nanya", "830B": "Nanya", "04EF": "Team Group", "8A76": "Lexar", "1315": "CXMT",
}

// memoryVendor nom du fabricant : dmidecode donne parfois un code JEDEC
// (« 80AD000080AD ») au lieu du nom
func memoryVendor(m string) string {
	m = strings.TrimSpace(m)
	if len(m) >= 4 {
		if name, ok := jedecVendors[strings.ToUpper(m[:4])]; ok {
			return name
		}
	}
	switch strings.ToLower(m) {
	case "unknown", "not specified", "undefined":
		return ""
	}
	return m
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
