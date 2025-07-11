package preset

import "fmt"

var Registry = map[string]Preset{}
// RegisterPresets регистрирует все пресеты в реестре
func InitAllPresets() {
	RegisterAreaPresets()    // <- сначала зависимые
	RegisterAddressPresets() // <- потом основные
}

func EnsurePresetLoaded(name string) error {	
	if _, ok := Registry[name]; !ok {
		return fmt.Errorf("preset %q still not found", name)
	}
	return nil
}

func GetPreset(key string) (Preset, error) {
	p, ok := Registry[key]
	if !ok {
		return Preset{}, fmt.Errorf("preset %q not found", key)
	}
	return p, nil
}