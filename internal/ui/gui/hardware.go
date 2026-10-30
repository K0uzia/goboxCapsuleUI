package gui

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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
	if len(wifiInterfaces(sysNet)) == 0 {
		out["wifi"] = "Pas de Wi-Fi sur cette machine"
	}
	if len(bluetoothAdapters(sysBluetooth)) == 0 {
		out["bluetooth"] = "Pas de Bluetooth sur cette machine"
	}
	if len(modemNames("/")) == 0 {
		out["sim"] = "Non vu par le système : relancer pour indiquer s'il est présent"
	}
	if len(cardReaders()) == 0 {
		out["lecteur"] = "Non vu par le système : relancer pour indiquer s'il est présent"
	}
	if !hasBattery(sysPower) {
		out["batterie"] = "Pas de batterie sur cette machine"
		out["chargeur"] = "Pas de batterie sur cette machine"
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
	kinds    map[string]string // prise → USB 2.0, USB 3.x ou USB-C
	typeC    int               // ports de /sys/class/typec
	total    int
	fromACPI bool
}

// Types de prises USB, dans l'ordre de la fiche
const (
	usbKind2 = "USB 2.0"
	usbKind3 = "USB 3.x"
	usbKindC = "USB-C"
)

var usbKindOrder = []string{usbKind2, usbKind3, usbKindC}

// betterKind garde le type le plus capable : les deux moitiés d'une prise USB 3
// sont sur un bus 2.0 et un bus 3.0
func betterKind(a, b string) string {
	for _, k := range []string{usbKindC, usbKind3} {
		if a == k || b == k {
			return k
		}
	}
	return usbKind2
}

// portKind type d'un port sysfs : relié à un connecteur Type-C, ou vitesse de son
// concentrateur
func portKind(hub, port string) string {
	if _, err := os.Stat(filepath.Join(port, "connector")); err == nil {
		return usbKindC
	}
	if speed, _ := strconv.Atoi(readTrim(filepath.Join(hub, "speed"))); speed >= 5000 {
		// Une prise USB-A a ses deux moitiés (bus 2.0 et 3.0, reliées par peer). Une
		// prise accessible seulement côté 3.0 est une USB-C, dont la partie 2.0 passe
		// ailleurs (constaté sur un HP EliteBook 850 G3 : 3 USB-A et 1 USB-C)
		_, err := os.Lstat(filepath.Join(port, "peer"))
		if err != nil && readTrim(filepath.Join(port, "connect_type")) == "hotplug" {
			return usbKindC
		}
		return usbKind3
	}
	return usbKind2
}

// kindOf type d'une prise
func (p usbPorts) kindOf(key string) string {
	if k := p.kinds[key]; k != "" {
		return k
	}
	return usbKind2
}

// kindTotals nombre de prises par type. Si le système compte des ports Type-C
// sans dire lesquels, ils sont retirés des USB 3.
func (p usbPorts) kindTotals() map[string]int {
	seen := map[string]bool{}
	out := map[string]int{}
	for _, k := range p.group {
		if !seen[k] {
			seen[k] = true
			out[p.kindOf(k)]++
		}
	}
	if out[usbKindC] == 0 && p.typeC > 0 {
		n := min(p.typeC, out[usbKind3])
		out[usbKindC], out[usbKind3] = n, out[usbKind3]-n
	}
	return out
}

// readUSBPorts lit les ports ; dmiUSB (prises listées par le BIOS) ne sert que
// si le microprogramme ne renseigne aucun port
func readUSBPorts(dmiUSB int) usbPorts {
	p := usbPorts{group: map[string]string{}, possible: map[string]string{}, kinds: map[string]string{}}
	typeC, _ := filepath.Glob("/sys/class/typec/port?")
	p.typeC = len(typeC)
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
			p.kinds[key] = betterKind(p.kinds[key], portKind(hub, port))
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

// usbKernelError erreurs par lesquelles le noyau signale une prise qui ne marche pas
var usbKernelError = regexp.MustCompile(`\busb ((?:usb)?\d+[-.\w]*): .*(device descriptor read|device not accepting address|unable to enumerate|Cannot enable|over-?current|cannot reset)`)

// parseUSBErrors noms système (usb1-port3, 1-2, 1-4-port2) des erreurs USB du noyau
func parseUSBErrors(log string) []string {
	var out []string
	for _, line := range strings.Split(log, "\n") {
		if m := usbKernelError.FindStringSubmatch(line); m != nil {
			out = append(out, m[1])
		}
	}
	return out
}

// portOfError prise désignée par un nom du noyau : port (usb1-port3) ou périphérique (1-2)
func (p usbPorts) portOfError(name string) string {
	for _, m := range []map[string]string{p.group, p.possible} {
		if k, ok := m[name]; ok {
			return k
		}
		if k := lookupPort(m, name); k != "" {
			return k
		}
	}
	return ""
}

// overCurrentPorts prises en surintensité avant même le test
func (p usbPorts) overCurrentPorts() []string {
	var out []string
	files, _ := filepath.Glob(filepath.Join(sysUSB, "*", "*:1.0", "*-port*", "over_current_count"))
	for _, f := range files {
		if n, _ := strconv.Atoi(readTrim(f)); n > 0 {
			if k := p.portOfError(filepath.Base(filepath.Dir(f))); k != "" {
				out = append(out, k)
			}
		}
	}
	return out
}

// kernelLog journal du noyau (dmesg, droits root) ; vide si illisible
func kernelLog() string {
	tool := toolPath("dmesg")
	if tool == "" {
		return ""
	}
	out, err := exec.Command(tool).Output()
	if err != nil {
		return ""
	}
	return string(out)
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

// physicalPorts prises de la machine. La liste du BIOS (DMI type 8) décrit la
// carte mère, puis la table Intel (vbt.go). Sans elles, les connecteurs d'une carte
// AMD ou NVIDIA sont fiables ; pas ceux d'Intel : un même processeur déclare les
// mêmes sur tout PC.
type physicalPorts struct {
	Video    map[string]int // type → nombre de prises ; vide : nombre inconnu
	USBC     int            // prises USB-C listées par le BIOS
	Why      string         // pourquoi le nombre est inconnu
	USB      int            // < 0 : nombre inconnu
	USBKinds map[string]int // type de prise USB → nombre
}

// toolPath chemin d'un outil système : /usr/sbin n'est pas toujours dans le PATH
// (Mint, ou lancé sans sudo)
func toolPath(name string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	for _, d := range []string{"/usr/sbin", "/sbin", "/usr/local/sbin"} {
		if p := filepath.Join(d, name); readable(p) {
			return p
		}
	}
	return ""
}

func readable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// readPhysicalPorts lit les prises ; dmidecode demande les droits root
func readPhysicalPorts() physicalPorts {
	var video map[string]int
	var usb, usbC int
	why := ""
	loadTypeCDriver()
	switch tool := toolPath("dmidecode"); {
	case tool == "":
		why = "dmidecode n'est pas installé"
	case !isAdmin():
		why = "droits administrateur manquants"
	default:
		out, err := exec.Command(tool, "-t", "8").Output()
		if err != nil {
			why = "dmidecode : " + err.Error()
		} else {
			video, usb, usbC = parseDMIConnectors(string(out))
			why = "le BIOS ne liste pas les prises"
		}
	}
	if len(video) == 0 {
		video = readVBTPorts()
	}
	if len(video) == 0 {
		video = gpuPorts(videoConnectors(), cardDrivers())
	}
	ports := readUSBPorts(usb)
	ports.typeC = max(ports.typeC, usbC)
	return physicalPorts{Video: video, USBC: usbC, Why: why, USB: ports.total, USBKinds: ports.kindTotals()}
}

// loadTypeCDriver charge le pilote qui révèle les prises USB-C (en mémoire seulement,
// rien n'est écrit sur le disque) : sans lui, /sys/class/typec reste vide
func loadTypeCDriver() {
	tool := toolPath("modprobe")
	if tool == "" || !isAdmin() || simulated {
		return
	}
	for _, mod := range []string{"typec_ucsi", "ucsi_acpi"} {
		_ = exec.Command(tool, "-q", mod).Run()
	}
}

// cardDrivers pilote de chaque carte graphique (card0 → i915)
func cardDrivers() map[string]string {
	out := map[string]string{}
	cards, _ := filepath.Glob("/sys/class/drm/card?")
	for _, c := range cards {
		if link, err := os.Readlink(filepath.Join(c, "device", "driver")); err == nil {
			out[filepath.Base(c)] = filepath.Base(link)
		}
	}
	return out
}

// gpuPorts prises d'après les cartes graphiques ; vide dès qu'une carte Intel
// est présente, ses connecteurs ne décrivent pas la machine
func gpuPorts(conns []videoConnector, drivers map[string]string) map[string]int {
	if len(drivers) == 0 {
		return nil
	}
	for _, d := range drivers {
		if d == "i915" || d == "xe" {
			return nil
		}
	}
	out := map[string]int{}
	for _, c := range conns {
		if c.Type != internalScreen {
			out[c.Type]++
		}
	}
	return out
}

var (
	dmiDP   = regexp.MustCompile(`displayport|display port|\b(mini[ -]?)?dp\d*\b`)
	dmiVGA  = regexp.MustCompile(`vga|d-?sub|db-15|\bcrt\b`)
	dmiUSBC = regexp.MustCompile(`usb[ -]?c\b|type[ -]?c|thunderbolt|\btbt\d*\b|usb4`)
)

// parseDMIConnectors compte les prises externes dans la sortie de dmidecode -t 8
func parseDMIConnectors(out string) (video map[string]int, usb, usbC int) {
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
		case dmiUSBC.MatchString(desc):
			usb++
			usbC++
		case values["Port Type"] == "usb" || strings.Contains(ext, "usb"):
			usb++
		}
	}
	return video, usb, usbC
}

// ---- Connecteurs vidéo ----------------------------------------------------

// videoConnector connecteur déclaré par la carte graphique
type videoConnector struct {
	Name      string // HDMI-A-1, DP-2…
	Type      string // HDMI, DisplayPort, VGA, DVI, Écran intégré
	Plug      string // prise réellement utilisée : VGA pour un écran analogique sur DP
	Connected bool
	Mode      string // résolution de l'écran branché (1920x1080)
}

// videoTypes types de sorties externes, dans l'ordre de la fiche
var videoTypes = []string{"HDMI", "DisplayPort", "VGA", "DVI", videoAny}

// videoAny sortie dont le type n'est connu qu'au branchement
const videoAny = "Sortie vidéo"

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

// edidAnalog vrai si l'écran se déclare analogique (octet 20 de l'EDID, bit 7 à
// 0) : un VGA relié par un pont interne est vu comme DisplayPort par la carte
func edidAnalog(edid []byte) bool {
	magic := []byte{0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0}
	return len(edid) >= 128 && bytes.Equal(edid[:8], magic) && edid[20]&0x80 == 0
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
		c := videoConnector{Name: name, Type: typ, Plug: typ, Connected: readTrim(filepath.Join(d, "status")) == "connected"}
		if c.Connected {
			c.Mode, _, _ = strings.Cut(readTrim(filepath.Join(d, "modes")), "\n")
			edid, _ := os.ReadFile(filepath.Join(d, "edid"))
			if typ != internalScreen && edidAnalog(edid) {
				c.Plug = "VGA"
			}
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
