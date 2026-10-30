package gui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Sur une image live, la mise à jour vise le système installé sur le disque du PC :
// celui de la session disparaîtrait au redémarrage. gobox le monte, lance la mise à
// jour dedans (chroot), puis démonte tout.

// blockDevice entrée de `lsblk -J`
type blockDevice struct {
	Name       string        `json:"name"`
	Path       string        `json:"path"`
	FSType     string        `json:"fstype"`
	Type       string        `json:"type"`
	Mountpoint string        `json:"mountpoint"`
	RM         bool          `json:"rm"`
	Children   []blockDevice `json:"children"`
}

// rootCandidates partitions qui peuvent porter un système Linux : disques internes,
// non montées, système de fichiers Linux ; encrypted compte les partitions chiffrées
func rootCandidates(lsblkJSON []byte) (paths []string, fstypes map[string]string, encrypted int, err error) {
	var doc struct {
		Devices []blockDevice `json:"blockdevices"`
	}
	if err := json.Unmarshal(lsblkJSON, &doc); err != nil {
		return nil, nil, 0, err
	}
	fstypes = map[string]string{}
	var walk func(d blockDevice, removable bool)
	walk = func(d blockDevice, removable bool) {
		removable = removable || d.RM
		switch {
		case removable || d.Type == "loop" || d.Type == "rom":
			return
		case d.FSType == "crypto_LUKS":
			encrypted++
		case d.Mountpoint == "" && (d.FSType == "ext4" || d.FSType == "xfs" || d.FSType == "btrfs"):
			paths = append(paths, d.Path)
			fstypes[d.Path] = d.FSType
		}
		for _, c := range d.Children {
			walk(c, removable)
		}
	}
	for _, d := range doc.Devices {
		walk(d, false)
	}
	return paths, fstypes, encrypted, nil
}

// osIdentity nom et gestionnaire de paquets du système dans root ; vide si ce n'en
// est pas un (partition de données, /home séparé)
func osIdentity(root string) (name, manager string) {
	for _, m := range []string{"apt-get", "dnf"} {
		if readable(filepath.Join(root, "usr/bin", m)) {
			manager = m
			break
		}
	}
	data, err := os.ReadFile(filepath.Join(root, "etc/os-release"))
	if err != nil || manager == "" {
		return "", ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if v, ok := strings.CutPrefix(line, "PRETTY_NAME="); ok {
			return strings.Trim(v, `"`), manager
		}
	}
	return "Linux", manager
}

// fstabMount montage à reprendre avant la mise à jour
type fstabMount struct{ Source, Target, FSType, Options string }

// bootMounts /boot et /boot/efi du fstab : le noyau mis à jour s'y écrit
func bootMounts(fstab string) []fstabMount {
	var out []fstabMount
	for _, line := range strings.Split(fstab, "\n") {
		f := strings.Fields(line)
		if len(f) < 3 || strings.HasPrefix(f[0], "#") {
			continue
		}
		if f[1] == "/boot" || f[1] == "/boot/efi" {
			m := fstabMount{Source: f[0], Target: f[1], FSType: f[2]}
			if len(f) > 3 {
				m.Options = f[3]
			}
			out = append(out, m)
		}
	}
	return out
}

// btrfsSubvols sous-volumes racine les plus courants, puis la racine du volume
var btrfsSubvols = []string{"@", "root", ""}

// diskSystem système Linux trouvé sur le disque
type diskSystem struct {
	Device, FSType, Subvol, Name, Manager string
}

func mountArgs(s diskSystem, opts, target string) []string {
	if s.Subvol != "" {
		if opts != "" {
			opts += ","
		}
		opts += "subvol=" + s.Subvol
	}
	args := []string{"mount"}
	if opts != "" {
		args = append(args, "-o", opts)
	}
	return append(args, s.Device, target)
}

func runQuiet(ctx context.Context, args ...string) error {
	path := toolPath(args[0])
	if path == "" {
		return fmt.Errorf("%s absent", args[0])
	}
	out, err := exec.CommandContext(ctx, path, args[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s : %v %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// findDiskSystems monte chaque partition en lecture seule pour y chercher un système ;
// encrypted compte les partitions chiffrées, illisibles
func findDiskSystems(ctx context.Context, logf func(string, ...any)) (systems []diskSystem, encrypted int) {
	_ = runQuiet(ctx, "vgchange", "-ay") // volumes LVM
	tool := toolPath("lsblk")
	if tool == "" {
		logf("lsblk absent : disques illisibles")
		return nil, 0
	}
	out, err := exec.CommandContext(ctx, tool, "-J", "-o", "NAME,PATH,FSTYPE,TYPE,MOUNTPOINT,RM").Output()
	if err != nil {
		logf("lsblk : %v", err)
		return nil, 0
	}
	paths, fstypes, encrypted, err := rootCandidates(out)
	if err != nil {
		logf("lsblk : %v", err)
		return nil, 0
	}
	dir, err := os.MkdirTemp("", "gobox-disque-")
	if err != nil {
		logf("Dossier temporaire : %v", err)
		return nil, encrypted
	}
	defer os.Remove(dir)
	for _, p := range paths {
		subvols := []string{""}
		if fstypes[p] == "btrfs" {
			subvols = btrfsSubvols
		}
		for _, sub := range subvols {
			s := diskSystem{Device: p, FSType: fstypes[p], Subvol: sub}
			if runQuiet(ctx, mountArgs(s, "ro", dir)...) != nil {
				continue
			}
			s.Name, s.Manager = osIdentity(dir)
			_ = runQuiet(ctx, "umount", dir)
			if s.Manager != "" {
				logf("Système trouvé : %s sur %s", s.Name, p)
				systems = append(systems, s)
				break
			}
		}
	}
	return systems, encrypted
}

// mountedSystem système monté pour la mise à jour
type mountedSystem struct {
	root           string
	resolvBackedUp bool
}

// mountForUpdate monte le système et ce dont la mise à jour a besoin
func mountForUpdate(ctx context.Context, s diskSystem) (*mountedSystem, error) {
	dir, err := os.MkdirTemp("", "gobox-disque-")
	if err != nil {
		return nil, err
	}
	m := &mountedSystem{root: dir}
	if err := runQuiet(ctx, mountArgs(s, "", dir)...); err != nil {
		os.Remove(dir)
		return nil, err
	}
	fstab, _ := os.ReadFile(filepath.Join(dir, "etc/fstab"))
	for _, b := range bootMounts(string(fstab)) {
		args := []string{"mount"}
		if b.FSType != "" && b.FSType != "auto" {
			args = append(args, "-t", b.FSType)
		}
		if err := runQuiet(ctx, append(args, b.Source, filepath.Join(dir, b.Target))...); err != nil {
			m.unmount()
			return nil, fmt.Errorf("%s : %w", b.Target, err)
		}
	}
	for _, sys := range []string{"proc", "sys", "dev", "dev/pts", "run"} {
		if err := runQuiet(ctx, "mount", "--bind", "/"+sys, filepath.Join(dir, sys)); err != nil {
			m.unmount()
			return nil, err
		}
	}
	// Le réseau de la session sert à la mise à jour ; le fichier d'origine revient après
	resolv := filepath.Join(dir, "etc/resolv.conf")
	if data, err := os.ReadFile("/etc/resolv.conf"); err == nil {
		if os.Rename(resolv, resolv+".gobox") == nil {
			m.resolvBackedUp = true
		}
		_ = os.WriteFile(resolv, data, 0o644)
	}
	return m, nil
}

// unmount remet le disque en l'état : resolv.conf d'origine, tout démonté
func (m *mountedSystem) unmount() error {
	resolv := filepath.Join(m.root, "etc/resolv.conf")
	if m.resolvBackedUp {
		_ = os.Remove(resolv)
		_ = os.Rename(resolv+".gobox", resolv)
	}
	bg := context.Background()
	_ = runQuiet(bg, "sync")
	err := runQuiet(bg, "umount", "-R", m.root)
	if err != nil {
		err = runQuiet(bg, "umount", "-R", "-l", m.root)
	}
	if err == nil {
		os.Remove(m.root)
	}
	return err
}
