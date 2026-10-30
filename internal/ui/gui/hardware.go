package gui

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Lecture directe de /sys pour ce que les tests et la fiche ont besoin de
// savoir sur le matériel présent : ports physiques, connecteurs vidéo,
// composants absents (un test sans objet ne peut pas être inventé).

const sysUSB = "/sys/bus/usb/devices"

// ---- Matériel absent ------------------------------------------------------

// absentHardware tests sans objet sur cette machine (ID du test → raison),
// d'après ce que le système détecte. Le technicien peut toujours relancer
// le test s'il sait que le composant est là.
func absentHardware() map[string]string {
	out := map[string]string{}
	if simulated {
		return out
	}
	if !hasWebcam() {
		out["webcam"] = "Aucune caméra détectée sur cette machine"
	}
	if !hasCardReader() {
		out["sd"] = "Aucun lecteur de carte détecté sur cette machine"
	}
	if len(pciDevices("0x0403")) == 0 && !usbHasInterface("01") {
		out["son"] = "Aucune carte son détectée sur cette machine"
	}
	kinds := internalDiskKinds()
	for id, label := range kindNames {
		if kinds[id] == 0 {
			out[id] = "Aucun " + label + " interne détecté"
		}
	}
	return out
}

func hasWebcam() bool {
	cams, _ := filepath.Glob("/dev/video*")
	return len(cams) > 0 || usbHasInterface("0e") // classe USB « vidéo »
}

// hasCardReader lecteur SD intégré (PCI : mmc_host) ou lecteur USB interne
func hasCardReader() bool {
	if hosts, _ := filepath.Glob("/sys/class/mmc_host/*"); len(hosts) > 0 {
		return true
	}
	for _, name := range usbDevices() {
		if n := strings.ToLower(name); strings.Contains(n, "card reader") || strings.Contains(n, "cardreader") {
			return true
		}
	}
	return false
}

// usbHasInterface vrai si une interface USB est de la classe donnée (hexa)
func usbHasInterface(class string) bool {
	files, _ := filepath.Glob(filepath.Join(sysUSB, "*", "bInterfaceClass"))
	for _, f := range files {
		if readTrim(f) == class {
			return true
		}
	}
	return false
}

// pciDevices adresses des périphériques PCI dont la classe commence par prefix
func pciDevices(prefix string) []string {
	var out []string
	dirs, _ := filepath.Glob("/sys/bus/pci/devices/*")
	for _, d := range dirs {
		if strings.HasPrefix(readTrim(filepath.Join(d, "class")), prefix) {
			out = append(out, filepath.Base(d))
		}
	}
	return out
}

// pciName nom lisible d'un périphérique PCI (lspci), vide si inconnu
func pciName(slot string) string {
	out, err := exec.Command("lspci", "-s", slot).Output()
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(out))
	// « 00:1f.3 Audio device: Intel Corporation … » → après le deuxième « : »
	if _, rest, ok := strings.Cut(line, " "); ok {
		if _, name, ok := strings.Cut(rest, ": "); ok {
			return shortVendor(name)
		}
	}
	return ""
}

var pciRevision = regexp.MustCompile(`\s*\(rev [0-9a-f]+\)$`)

// shortVendor nom sans révision ni forme juridique : « Realtek
// Semiconductor Co., Ltd. RTL8111… (rev 15) » → « Realtek RTL8111… »
func shortVendor(name string) string {
	name = pciRevision.ReplaceAllString(name, "")
	for _, s := range []string{" Semiconductor Co., Ltd.", " Corporation", " Co., Ltd.", ", Inc.", " Inc."} {
		name = strings.Replace(name, s, "", 1)
	}
	return strings.TrimSpace(name)
}

// internalDiskKinds nombre de disques internes par type (hdd, ssd, nvme),
// sans les clés USB, lecteurs optiques, cartes SD et disques virtuels
func internalDiskKinds() map[string]int {
	out := map[string]int{}
	for _, d := range internalDisks() {
		out[d.kind]++
	}
	return out
}

type blockDisk struct {
	name, kind string
}

func internalDisks() []blockDisk {
	var out []blockDisk
	dirs, _ := filepath.Glob("/sys/block/*")
	for _, d := range dirs {
		name := filepath.Base(d)
		for _, p := range []string{"loop", "ram", "zram", "sr", "mmcblk", "dm-", "md", "fd", "nbd"} {
			if strings.HasPrefix(name, p) {
				name = ""
			}
		}
		if name == "" || readTrim(filepath.Join(d, "removable")) == "1" {
			continue
		}
		if real, err := filepath.EvalSymlinks(d); err == nil && strings.Contains(real, "/usb") {
			continue
		}
		kind := "ssd"
		switch {
		case strings.HasPrefix(name, "nvme"):
			kind = "nvme"
		case readTrim(filepath.Join(d, "queue", "rotational")) == "1":
			kind = "hdd"
		}
		out = append(out, blockDisk{name, kind})
	}
	return out
}

// ---- Ports USB physiques --------------------------------------------------

// usbPorts ports USB de la machine. Le contrôleur en déclare plus qu'il n'y a
// de prises : seul le microprogramme (ACPI, connect_type) dit lesquelles sont
// accessibles. Un port USB 3 apparaît deux fois (bus USB 2 et bus USB 3, reliés
// par « peer ») : il ne compte qu'une fois. total < 0 : nombre inconnu.
type usbPorts struct {
	group    map[string]string // nom du port sysfs (usb1-port3, 1-4-port2) → prise
	possible map[string]string // type non renseigné : prise seulement si on y branche
	total    int
	fromACPI bool
}

// readUSBPorts lit les ports ; dmiUSB (prises listées par le BIOS) ne sert que
// si le microprogramme ne renseigne aucun port
func readUSBPorts(dmiUSB int) usbPorts {
	p := usbPorts{group: map[string]string{}, possible: map[string]string{}}
	// concentrateurs : contrôleurs (usbN), et hubs branchés sur un port câblé
	hubs, _ := filepath.Glob(filepath.Join(sysUSB, "usb*"))
	for len(hubs) > 0 {
		hub := hubs[0]
		hubs = hubs[1:]
		base := filepath.Base(hub)
		ports, _ := filepath.Glob(filepath.Join(hub, base+":1.0", "*-port*"))
		if strings.HasPrefix(base, "usb") {
			ports, _ = filepath.Glob(filepath.Join(hub, "*-0:1.0", "*-port*"))
		}
		for _, port := range ports {
			name := filepath.Base(port)
			key := name
			if peer, err := os.Readlink(filepath.Join(port, "peer")); err == nil && filepath.Base(peer) < key {
				key = filepath.Base(peer)
			}
			switch readTrim(filepath.Join(port, "connect_type")) {
			case "not used":
			case "hardwired":
				// un hub interne derrière ce port porte peut-être les prises
				if dev := usbDeviceOnPort(name); dev != "" && readTrim(filepath.Join(sysUSB, dev, "bDeviceClass")) == "09" {
					hubs = append(hubs, filepath.Join(sysUSB, dev))
				}
			case "hotplug":
				p.group[name] = key
			default:
				p.possible[name] = key
			}
		}
	}
	p.finish(dmiUSB)
	return p
}

// finish compte les prises : la moitié USB 3 non renseignée d'une prise connue
// en fait partie
func (p *usbPorts) finish(dmiUSB int) {
	seen := map[string]bool{}
	for _, k := range p.group {
		seen[k] = true
	}
	for name, k := range p.possible {
		if seen[k] {
			p.group[name] = k
			delete(p.possible, name)
		}
	}
	p.total, p.fromACPI = len(seen), len(seen) > 0
	if !p.fromACPI {
		p.total = -1
		if dmiUSB > 0 {
			p.total = dmiUSB
		}
	}
}

// adopt un périphérique vient d'être branché sur un port non renseigné : c'est
// donc une prise. Rend la prise, "" si le port reste inconnu.
func (p *usbPorts) adopt(dev string) string {
	port := lookupPort(p.possible, dev)
	if port == "" {
		return ""
	}
	for name, k := range p.possible {
		if k == port {
			p.group[name] = k
			delete(p.possible, name)
		}
	}
	if p.fromACPI {
		p.total++
	}
	return port
}

// usbDeviceOnPort périphérique branché sur un port (usb1-port3 → 1-3)
func usbDeviceOnPort(port string) string {
	name, num, ok := strings.Cut(port, "-port")
	if !ok {
		return ""
	}
	if bus, found := strings.CutPrefix(name, "usb"); found {
		return bus + "-" + num
	}
	return name + "." + num
}

var usbDevName = regexp.MustCompile(`^(\d+)-([\d.]+)$`)

// portOf port physique sur lequel est branché un périphérique (1-3.2 :
// port 3 du contrôleur 1, ou port 2 du hub interne 1-3), "" si inconnu
func (p usbPorts) portOf(dev string) string { return lookupPort(p.group, dev) }

func lookupPort(ports map[string]string, dev string) string {
	m := usbDevName.FindStringSubmatch(dev)
	if m == nil {
		return ""
	}
	bus, chain := m[1], strings.Split(m[2], ".")
	for i := range chain {
		parent := "usb" + bus
		if i > 0 {
			parent = bus + "-" + strings.Join(chain[:i], ".")
		}
		if g, ok := ports[parent+"-port"+chain[i]]; ok {
			return g
		}
	}
	return ""
}

// usbDevices périphériques USB présents (nom sysfs, ex. 1-3.2 → nom lisible)
func usbDevices() map[string]string {
	devs := map[string]string{}
	dirs, _ := filepath.Glob(filepath.Join(sysUSB, "*"))
	for _, d := range dirs {
		base := filepath.Base(d)
		if !usbDevName.MatchString(base) {
			continue // hub racine (usbN) ou interface (1-3:1.0)
		}
		name := readTrim(filepath.Join(d, "product"))
		if name == "" {
			name = readTrim(filepath.Join(d, "idVendor")) + ":" + readTrim(filepath.Join(d, "idProduct"))
		}
		if m := readTrim(filepath.Join(d, "manufacturer")); m != "" && !strings.Contains(name, m) {
			name = m + " " + name
		}
		devs[base] = name
	}
	return devs
}

// ---- Prises physiques (BIOS) ----------------------------------------------

// physicalPorts prises de la machine. La carte graphique ne les connaît pas :
// un même processeur déclare les mêmes connecteurs sur tout PC (3 DP + 3 HDMI,
// VGA par pont vu comme DP). La liste du BIOS (DMI type 8) décrit la carte mère.
type physicalPorts struct {
	Video map[string]int // type → nombre de prises ; vide : nombre inconnu
	USB   int            // < 0 : nombre inconnu
}

// readPhysicalPorts lit les prises ; dmidecode demande les droits root
func readPhysicalPorts() physicalPorts {
	var video map[string]int
	var usb int
	if out, err := exec.Command("dmidecode", "-t", "8").Output(); err == nil {
		video, usb = parseDMIConnectors(string(out))
	}
	return physicalPorts{Video: video, USB: readUSBPorts(usb).total}
}

var (
	dmiDP  = regexp.MustCompile(`displayport|display port|\b(mini[ -]?)?dp\d*\b`)
	dmiVGA = regexp.MustCompile(`vga|d-?sub|db-15|\bcrt\b`)
)

// parseDMIConnectors compte les prises externes dans la sortie de dmidecode -t 8
func parseDMIConnectors(out string) (video map[string]int, usb int) {
	video = map[string]int{}
	for _, block := range strings.Split(out, "\n\n") {
		values := map[string]string{}
		for _, line := range strings.Split(block, "\n") {
			if k, v, ok := strings.Cut(strings.TrimSpace(line), ": "); ok {
				values[k] = strings.ToLower(v)
			}
		}
		ext := values["External Connector Type"]
		if ext == "" || ext == "none" {
			continue
		}
		desc := values["External Reference Designator"] + " " + values["Internal Reference Designator"] + " " + ext
		switch {
		case strings.Contains(desc, "hdmi"):
			video["HDMI"]++
		case dmiDP.MatchString(desc):
			video["DisplayPort"]++
		case strings.Contains(desc, "dvi"):
			video["DVI"]++
		case dmiVGA.MatchString(desc):
			video["VGA"]++
		case values["Port Type"] == "usb" || strings.Contains(ext, "usb"):
			usb++
		}
	}
	return video, usb
}

// ---- Connecteurs vidéo ----------------------------------------------------

// videoConnector connecteur déclaré par la carte graphique
type videoConnector struct {
	Name      string // HDMI-A-1, DP-2…
	Type      string // HDMI, DisplayPort, VGA, DVI, Écran intégré
	Connected bool
	Mode      string // résolution de l'écran branché (1920x1080)
}

// videoTypes types de sorties externes, dans l'ordre de la fiche
var videoTypes = []string{"HDMI", "DisplayPort", "VGA", "DVI"}

const internalScreen = "Écran intégré"

func connectorType(name string) string {
	switch {
	case strings.HasPrefix(name, "HDMI"):
		return "HDMI"
	case strings.HasPrefix(name, "DP"):
		return "DisplayPort"
	case strings.HasPrefix(name, "VGA"):
		return "VGA"
	case strings.HasPrefix(name, "DVI"):
		return "DVI"
	case strings.HasPrefix(name, "eDP"), strings.HasPrefix(name, "LVDS"), strings.HasPrefix(name, "DSI"):
		return internalScreen
	}
	return "" // Virtual, Writeback…
}

// videoConnectors connecteurs de toutes les cartes graphiques, triés
func videoConnectors() []videoConnector {
	var out []videoConnector
	dirs, _ := filepath.Glob("/sys/class/drm/card*-*")
	for _, d := range dirs {
		_, name, _ := strings.Cut(filepath.Base(d), "-") // card0-HDMI-A-1 → HDMI-A-1
		typ := connectorType(name)
		if typ == "" {
			continue
		}
		c := videoConnector{Name: name, Type: typ, Connected: readTrim(filepath.Join(d, "status")) == "connected"}
		if c.Connected {
			c.Mode, _, _ = strings.Cut(readTrim(filepath.Join(d, "modes")), "\n")
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func readTrim(path string) string {
	b, _ := os.ReadFile(path)
	return strings.TrimSpace(string(b))
}
