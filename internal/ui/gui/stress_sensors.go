package gui

import (
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gobox/internal/probe"
)

// simulated vrai hors Linux : les tests prennent leurs mesures d'une source
// simulée, signalée à l'écran, pour travailler les interfaces sans le matériel
var simulated = runtime.GOOS != "linux"

// stressSample un relevé pendant le stress test
type stressSample struct {
	Temp     float64 // °C
	HasTemp  bool
	Freq     float64 // GHz, moyenne des cœurs
	Load     float64 // %, tous cœurs
	CoreLoad []float64
	CoreFreq []float64 // GHz
	Power    float64   // W, paquet CPU
	HasPower bool
}

// stressSensors fournit les relevés ; burns indique s'il faut lancer la
// vraie charge (non en simulation)
type stressSensors interface {
	sample() stressSample
	burns() bool
}

func newStressSensors() stressSensors {
	if simulated {
		return newSimSensors()
	}
	return newLinuxSensors()
}

// ---- Mesures réelles (Linux) ----------------------------------------------

type linuxSensors struct {
	prevCores  []cpuTimes
	prevEnergy float64 // µJ
	prevAt     time.Time
}

type cpuTimes struct{ idle, total uint64 }

func newLinuxSensors() *linuxSensors {
	l := &linuxSensors{}
	l.prevCores = readCoreTimes()
	l.prevEnergy, _ = readEnergy()
	l.prevAt = time.Now()
	return l
}

func (l *linuxSensors) burns() bool { return true }

func (l *linuxSensors) sample() stressSample {
	var s stressSample
	if t, err := probe.CPUTemperature(); err == nil {
		s.Temp, s.HasTemp = t, true
	}

	cores := readCoreTimes()
	for i := range cores {
		if i < len(l.prevCores) {
			s.CoreLoad = append(s.CoreLoad, loadBetween(l.prevCores[i], cores[i]))
		}
	}
	l.prevCores = cores
	for _, v := range s.CoreLoad {
		s.Load += v
	}
	if len(s.CoreLoad) > 0 {
		s.Load /= float64(len(s.CoreLoad))
	}

	for i := range s.CoreLoad {
		f := filepath.Join("/sys/devices/system/cpu", "cpu"+strconv.Itoa(i), "cpufreq", "scaling_cur_freq")
		v, _ := readFloat(f)
		s.CoreFreq = append(s.CoreFreq, v/1e6)
		s.Freq += v / 1e6
	}
	if len(s.CoreFreq) > 0 {
		s.Freq /= float64(len(s.CoreFreq))
	}

	// RAPL : compteur d'énergie du paquet CPU (lisible en root)
	if e, err := readEnergy(); err == nil && l.prevEnergy > 0 && e >= l.prevEnergy {
		dt := time.Since(l.prevAt).Seconds()
		if dt > 0 {
			s.Power, s.HasPower = (e-l.prevEnergy)/1e6/dt, true
		}
		l.prevEnergy = e
	}
	l.prevAt = time.Now()
	return s
}

// readCoreTimes compteurs de chaque ligne « cpuN » de /proc/stat
func readCoreTimes() []cpuTimes {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return nil
	}
	var out []cpuTimes
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 || !strings.HasPrefix(fields[0], "cpu") || fields[0] == "cpu" {
			continue
		}
		var t cpuTimes
		for i, f := range fields[1:] {
			v, _ := strconv.ParseUint(f, 10, 64)
			t.total += v
			if i == 3 || i == 4 { // idle + iowait
				t.idle += v
			}
		}
		out = append(out, t)
	}
	return out
}

func loadBetween(a, b cpuTimes) float64 {
	total := b.total - a.total
	if total == 0 {
		return 0
	}
	return 100 * float64(total-(b.idle-a.idle)) / float64(total)
}

func readEnergy() (float64, error) {
	return readFloat("/sys/class/powercap/intel-rapl:0/energy_uj")
}

func readFloat(path string) (float64, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(strings.TrimSpace(string(b)), 64)
}

// ---- Mesures simulées -----------------------------------------------------

// simSensors machine fictive à 8 threads : la température monte vers un
// palier tiré au hasard ; au-delà de 86 °C la fréquence baisse (throttling)
type simSensors struct {
	start   time.Time
	idle    float64
	plateau float64
	maxFreq float64
	cores   int
	rng     *rand.Rand
}

func newSimSensors() *simSensors {
	r := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 7))
	return &simSensors{
		start: time.Now(), idle: 41, plateau: 72 + r.Float64()*20,
		maxFreq: 3.4, cores: 8, rng: r,
	}
}

func (s *simSensors) burns() bool { return false }

func (s *simSensors) sample() stressSample {
	t := time.Since(s.start).Seconds()
	temp := s.idle + (s.plateau-s.idle)*(1-math.Exp(-t/35)) + s.rng.NormFloat64()*0.6
	freq := s.maxFreq
	if temp > 86 {
		freq -= (temp - 86) * 0.12 // throttling
	}
	out := stressSample{Temp: temp, HasTemp: true, Power: 14 + 0.3*(temp-s.idle) + s.rng.Float64(), HasPower: true}
	for i := 0; i < s.cores; i++ {
		l := 97 + s.rng.Float64()*3
		f := freq - s.rng.Float64()*0.08
		out.CoreLoad = append(out.CoreLoad, l)
		out.CoreFreq = append(out.CoreFreq, f)
		out.Load += l / float64(s.cores)
		out.Freq += f / float64(s.cores)
	}
	return out
}
