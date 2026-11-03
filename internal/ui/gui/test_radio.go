package gui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Wi-Fi, Bluetooth, modem SIM et lecteur de puce : l'atelier n'a pas toujours de
// carte à insérer, on vérifie donc que le matériel est présent et reconnu par le
// système. « DÉTECTÉ » dit bien qu'aucune carte n'a été lue. Absent : la ligne ne
// paraît pas sur la fiche. Rien à demander au technicien : lancés à l'ouverture.

const (
	sysNet       = "/sys/class/net"
	sysRfkill    = "/sys/class/rfkill"
	sysBluetooth = "/sys/class/bluetooth"
	pillDetected = "DÉTECTÉ"
)

// rfkillState radio coupée par logiciel ou par l'interrupteur ou la touche du PC
func rfkillState(root, typ string) (soft, hard bool) {
	dirs, _ := filepath.Glob(filepath.Join(root, "rfkill*"))
	for _, d := range dirs {
		if readTrim(filepath.Join(d, "type")) == typ {
			soft = soft || readTrim(filepath.Join(d, "soft")) == "1"
			hard = hard || readTrim(filepath.Join(d, "hard")) == "1"
		}
	}
	return soft, hard
}

// blockedDetail texte si la radio est coupée, vide sinon
func blockedDetail(soft, hard bool) string {
	switch {
	case hard:
		return "coupé par l'interrupteur ou la touche du PC"
	case soft:
		return "désactivé dans le système"
	}
	return ""
}

// wifiInterfaces cartes Wi-Fi
func wifiInterfaces(root string) []string {
	var out []string
	dirs, _ := filepath.Glob(filepath.Join(root, "*"))
	for _, d := range dirs {
		if _, err := os.Stat(filepath.Join(d, "wireless")); err == nil {
			out = append(out, filepath.Base(d))
		} else if _, err := os.Stat(filepath.Join(d, "phy80211")); err == nil {
			out = append(out, filepath.Base(d))
		}
	}
	return out
}

// bluetoothAdapters adaptateurs hciN (sans les appareils associés hci0:1)
func bluetoothAdapters(root string) []string {
	var out []string
	dirs, _ := filepath.Glob(filepath.Join(root, "hci*"))
	for _, d := range dirs {
		if name := filepath.Base(d); !strings.Contains(name, ":") {
			out = append(out, name)
		}
	}
	return out
}

// countNetworks lignes de réseaux dans la sortie de nmcli ou de iw
func countNetworks(tool, out string) int {
	n := 0
	for _, line := range strings.Split(out, "\n") {
		if tool == "iw" {
			if strings.HasPrefix(line, "BSS ") {
				n++
			}
		} else if strings.TrimSpace(line) != "" {
			n++
		}
	}
	return n
}

// scanWifi compte les réseaux visibles ; ok faux si aucun outil ne peut balayer
func scanWifi(ctx context.Context, iface string) (count int, ok bool) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if tool := toolPath("nmcli"); tool != "" {
		out, err := exec.CommandContext(ctx, tool, "-t", "-f", "BSSID", "dev", "wifi", "list", "ifname", iface, "--rescan", "yes").Output()
		if err == nil {
			return countNetworks("nmcli", string(out)), true
		}
	}
	tool := toolPath("iw")
	if tool == "" || !isAdmin() {
		return 0, false
	}
	if ip := toolPath("ip"); ip != "" {
		_ = exec.CommandContext(ctx, ip, "link", "set", iface, "up").Run()
	}
	out, err := exec.CommandContext(ctx, tool, "dev", iface, "scan").Output()
	if err != nil {
		return 0, false
	}
	return countNetworks("iw", string(out)), true
}

func wifiScan(ctx context.Context, logf func(string, ...any)) result {
	ifaces := wifiInterfaces(sysNet)
	if len(ifaces) == 0 {
		logf("Aucune carte Wi-Fi")
		return result{Status: statusNA, Detail: "Pas de Wi-Fi"}
	}
	name := orUnknown(netChip(ifaces[0]), ifaces[0])
	logf("Carte Wi-Fi : %s", name)
	if why := blockedDetail(rfkillState(sysRfkill, "wlan")); why != "" {
		logf("Wi-Fi %s", why)
		return result{Status: statusWarn, Detail: name + " : " + why}
	}
	n, ok := scanWifi(ctx, ifaces[0])
	switch {
	case !ok:
		logf("Balayage des réseaux impossible")
		return result{Status: statusOK, Detail: name + " : détectée, balayage impossible", Data: map[string]string{"pill": pillDetected}}
	case n == 0:
		logf("Aucun réseau vu")
		return result{Status: statusWarn, Detail: name + " : aucun réseau vu"}
	}
	logf("%d réseau(x) vu(s)", n)
	return result{Status: statusOK, Detail: fmt.Sprintf("%s : %d réseau(x) vu(s)", name, n)}
}

var bluetoothDevice = regexp.MustCompile(`(?m)\bNEW\b.*\bDevice\b`)

func bluetoothScan(ctx context.Context, logf func(string, ...any)) result {
	adapters := bluetoothAdapters(sysBluetooth)
	if len(adapters) == 0 {
		logf("Aucun adaptateur Bluetooth")
		return result{Status: statusNA, Detail: "Pas de Bluetooth"}
	}
	logf("Adaptateur Bluetooth : %s", strings.Join(adapters, ", "))
	if why := blockedDetail(rfkillState(sysRfkill, "bluetooth")); why != "" {
		logf("Bluetooth %s", why)
		return result{Status: statusWarn, Detail: "Bluetooth " + why}
	}
	if tool := toolPath("bluetoothctl"); tool != "" {
		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		if out, err := exec.CommandContext(ctx, tool, "--timeout", "8", "scan", "on").Output(); err == nil {
			if n := len(bluetoothDevice.FindAll(out, -1)); n > 0 {
				logf("%d appareil(s) vu(s)", n)
				return result{Status: statusOK, Detail: fmt.Sprintf("%d appareil(s) vu(s)", n)}
			}
		}
	}
	return result{Status: statusOK, Detail: "Adaptateur reconnu", Data: map[string]string{"pill": pillDetected}}
}

// modemNames modem WWAN, qui porte le lecteur de carte SIM
func modemNames(root string) []string {
	var out []string
	for _, pattern := range []string{
		"sys/class/wwan/*", "sys/class/net/wwan*", "sys/class/net/wwp*", "dev/cdc-wdm*", "sys/bus/mhi/devices/*",
		"sys/bus/usb/drivers/qmi_wwan/*:*", "sys/bus/usb/drivers/cdc_mbim/*:*", "sys/bus/usb/drivers/option/*:*", "sys/bus/usb/drivers/sierra/*:*",
	} {
		found, _ := filepath.Glob(filepath.Join(root, pattern))
		for _, f := range found {
			out = append(out, filepath.Base(f))
		}
	}
	return append(out, usbModems(filepath.Join(root, "sys/bus/usb/devices"))...)
}

func simScan(_ context.Context, logf func(string, ...any)) result {
	modems := modemNames("/")
	if len(modems) == 0 {
		logf("Aucun modem : pas de lecteur SIM relié au système")
		return result{Status: statusNA, Detail: "Pas de modem SIM"}
	}
	logf("Modem : %s", modems[0])
	if why := blockedDetail(rfkillState(sysRfkill, "wwan")); why != "" {
		return result{Status: statusWarn, Detail: "Modem " + why}
	}
	return result{Status: statusOK, Detail: "Modem reconnu, carte SIM non testée", Data: map[string]string{"pill": pillDetected}}
}

// usbCardReaders lecteurs de puce USB : classe d'interface 0b (CCID), ou lecteurs des
// portables qui se déclarent autrement (classe du constructeur), reconnus par leur nom
func usbCardReaders(root string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(dev string) {
		if !seen[dev] {
			seen[dev] = true
			out = append(out, orUnknown(readTrim(filepath.Join(root, dev, "product")), "Lecteur USB"))
		}
	}
	files, _ := filepath.Glob(filepath.Join(root, "*", "bInterfaceClass"))
	for _, f := range files {
		if readTrim(f) == "0b" {
			// interface 1-4:1.0 : le périphérique est 1-4, à côté d'elle dans root
			dev, _, _ := strings.Cut(filepath.Base(filepath.Dir(f)), ":")
			add(dev)
		}
	}
	products, _ := filepath.Glob(filepath.Join(root, "*", "product"))
	for _, f := range products {
		if usbReaderName.MatchString(readTrim(f)) {
			add(filepath.Base(filepath.Dir(f)))
		}
	}
	return out
}

var (
	usbReaderName = regexp.MustCompile(`(?i)smart ?card|\bccid\b|controlvault|\bau9540\b|\bbcm58\d\d`)
	usbModemName  = regexp.MustCompile(`(?i)\bwwan\b|\blte\b|\bhspa\b|\bgobi\b|\bmbim\b|snapdragon|\bxmm\d|mobile broadband`)
)

// usbModems modems USB reconnus par leur nom, même sans pilote chargé
func usbModems(root string) []string {
	var out []string
	products, _ := filepath.Glob(filepath.Join(root, "*", "product"))
	for _, f := range products {
		if name := readTrim(f); usbModemName.MatchString(name) {
			out = append(out, name)
		}
	}
	return out
}

// usbInventory tout ce qui est branché en USB, pour le diagnostic
func usbInventory(root string) string {
	var b strings.Builder
	devs, _ := filepath.Glob(filepath.Join(root, "*", "idVendor"))
	for _, f := range devs {
		d := filepath.Dir(f)
		classes := map[string]bool{}
		ifaces, _ := filepath.Glob(filepath.Join(root, filepath.Base(d)+":*", "bInterfaceClass"))
		for _, i := range ifaces {
			classes[readTrim(i)] = true
		}
		var list []string
		for c := range classes {
			list = append(list, c)
		}
		fmt.Fprintf(&b, "  %s %s:%s classes %s : %s %s\n", filepath.Base(d), readTrim(filepath.Join(d, "idVendor")), readTrim(filepath.Join(d, "idProduct")),
			strings.Join(list, ","), readTrim(filepath.Join(d, "manufacturer")), readTrim(filepath.Join(d, "product")))
	}
	return b.String()
}

var pciSmartCard = regexp.MustCompile(`(?i)^\S+\s+(.*smart ?card.*)$`)

// parseSmartCardPCI lecteurs de puce dans la sortie de lspci
func parseSmartCardPCI(out string) []string {
	var found []string
	for _, line := range strings.Split(out, "\n") {
		if m := pciSmartCard.FindStringSubmatch(line); m != nil {
			found = append(found, shortVendor(m[1]))
		}
	}
	return found
}

func cardReaders() []string {
	found := usbCardReaders(sysUSB)
	if tool := toolPath("lspci"); tool != "" {
		if out, err := exec.Command(tool).Output(); err == nil {
			found = append(found, parseSmartCardPCI(string(out))...)
		}
	}
	return found
}

func readerScan(_ context.Context, logf func(string, ...any)) result {
	found := cardReaders()
	if len(found) == 0 {
		logf("Aucun lecteur de puce")
		return result{Status: statusNA, Detail: "Pas de lecteur de puce"}
	}
	logf("Lecteur de puce : %s", strings.Join(found, ", "))
	return result{Status: statusOK, Detail: found[0] + " : reconnu, carte non testée", Data: map[string]string{"pill": pillDetected}}
}

// autoScans tests lancés à l'ouverture de gobox : ils ne demandent rien
var autoScans = []struct {
	id   string
	scan func(context.Context, func(string, ...any)) result
}{{"wifi", wifiScan}, {"bluetooth", bluetoothScan}, {"sim", simScan}, {"lecteur", readerScan}}

// runScan test relancé à la main depuis son bouton
func runScan(scan func(context.Context, func(string, ...any)) result, simulatedDetail string) func(*testCtx) result {
	return func(t *testCtx) result {
		t.progress(-1)
		if simulated {
			t.markSimulated()
			return result{Status: statusOK, Detail: simulatedDetail}
		}
		return scan(t.ctx, t.log)
	}
}

const pillPresent = "PRÉSENT"

// runSlotScan lecteur de puce ou SIM : un emplacement peut exister sans que le système
// le voie (SIM sans modem installé, lecteur coupé dans le BIOS). S'il ne le voit pas,
// le technicien dit ce qu'il a sous les yeux.
func runSlotScan(what string, scan func(context.Context, func(string, ...any)) result, simulatedDetail string) func(*testCtx) result {
	auto := runScan(scan, simulatedDetail)
	return func(t *testCtx) result {
		r := auto(t)
		if t.cancelled() || r.Status != statusNA {
			return r
		}
		switch t.choose(what+" non vu par le système. Sur ce PC :",
			choice{"Présent, non testable", statusOK}, choice{"En panne", statusFail}, choice{"Absent", statusNA}) {
		case statusOK:
			return result{Status: statusOK, Detail: "Présent, non vu par le système", Data: map[string]string{"pill": pillPresent}}
		case statusFail:
			return result{Status: statusFail, Detail: "Présent, en panne"}
		case statusNA:
			return r
		}
		return result{}
	}
}
