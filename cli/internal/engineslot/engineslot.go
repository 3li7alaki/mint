package engineslot

import (
	"os/exec"

	"mint/internal/engine"
)

type Probe func(engine.Row) error

type Options struct {
	Slots []string
	Probe Probe
}

func ReferenceSlots() []string {
	return engine.ReferenceSlots()
}

func ResolveDefault(opts Options) (string, bool) {
	probe := opts.Probe
	if probe == nil {
		probe = LookPathProbe
	}
	slots := opts.Slots
	if len(slots) == 0 {
		slots = ReferenceSlots()
	}
	for _, slot := range slots {
		row, ok := engine.Registry[slot]
		if !ok {
			continue
		}
		if err := probe(row); err == nil {
			return row.Key, true
		}
	}
	return "", false
}

func LookPathProbe(row engine.Row) error {
	_, err := exec.LookPath(row.Binary)
	return err
}
