package probe

//
// import (
// 	"fmt"
// 	"os"
// )
//
// const (
// 	bluetoothRoot = "/sys/class/bluetooth/" // Périphériques Bluetooth
// 	hciRoot       = "/sys/class/net/hci"    // Adaptateurs HCI (hci0, hci1...)
// )
//
// type BluetoothAdapter struct {
// 	Name       string // Ex: "hci0"
// 	MACAddress string // Adresse MAC Bluetooth
// 	IsUp       bool   // État actif
// 	Type       string // "USB", "PCI", etc.
// 	Vendor     string // Fabricant
// }

// func ListBluetoothAdapters() ([]BluetoothAdapter, error) {
// 	// Logique pour lister les adaptateurs Bluetooth
// 	entries, err := os.ReadDir(bluetoothRoot)
// 	if err != nil {
// 		// Si le répertoire n'existe pas, pas de Bluetooth
// 		if os.IsNotExist(err) {
// 			return []BluetoothAdapter{}, nil
// 		}
// 		return nil, fmt.Errorf("lecture des adaptateurs Bluetooth: %w", err)
// 	}
//
// 	// ... traitement
// }
