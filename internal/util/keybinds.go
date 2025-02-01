package util

import (
	"log/slog"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
)

var (
	ModifiersInt = map[string]int{
		"lctrl":     gdk.KEY_Control_L,
		"rctrl":     gdk.KEY_Control_R,
		"lalt":      gdk.KEY_Alt_L,
		"ralt":      gdk.KEY_Alt_R,
		"lshift":    gdk.KEY_Shift_L,
		"rshift":    gdk.KEY_Shift_R,
		"shiftlock": gdk.KEY_Shift_Lock,
	}
	Modifiers = map[string]gdk.ModifierType{
		"ctrl":   gdk.ControlMask,
		"lctrl":  gdk.ControlMask,
		"rctrl":  gdk.ControlMask,
		"alt":    gdk.AltMask,
		"lalt":   gdk.AltMask,
		"ralt":   gdk.AltMask,
		"lshift": gdk.ShiftMask,
		"rshift": gdk.ShiftMask,
		"shift":  gdk.ShiftMask,
	}
	SpecialKeys = map[string]int{
		"backspace": int(gdk.KEY_BackSpace),
		"tab":       int(gdk.KEY_Tab),
		"esc":       int(gdk.KEY_Escape),
		"enter":     int(gdk.KEY_Return),
		"down":      int(gdk.KEY_Down),
		"up":        int(gdk.KEY_Up),
		"left":      int(gdk.KEY_Left),
		"right":     int(gdk.KEY_Right),
	}
)

type KeybindCommand struct {
	Label string `koanf:"label"`
	Key   string `koanf:"key"`
	Cmd   string `koanf:"cmd"`
}

type Keybinds map[int]map[gdk.ModifierType]KeybindCommand

func ParseKeybind(val string) (int, gdk.ModifierType) {
	fields := strings.Fields(val)

	m := []gdk.ModifierType{}

	key := 0

	for _, v := range fields {
		if len(v) > 1 {
			if val, exists := Modifiers[v]; exists {
				m = append(m, val)
			}

			if val, exists := SpecialKeys[v]; exists {
				key = val
			}
		} else {
			key = int(v[0])
		}
	}

	modifier := gdk.NoModifierMask

	switch len(m) {
	case 1:
		modifier = m[0]
	case 2:
		modifier = m[0] | m[1]
	case 3:
		modifier = m[0] | m[1] | m[2]
	}

	return key, modifier
}

func ValidateKeybind(bind string) bool {
	fields := strings.Fields(bind)

	for _, v := range fields {
		if len(v) > 1 {
			_, existsMod := Modifiers[v]
			_, existsSpecial := SpecialKeys[v]

			if !existsMod && !existsSpecial {
				slog.Error("keybinds", "bind", bind, "key", v)
				return false
			}
		}
	}

	return true
}

func (Keybinds) ValidateTriggerLabels(bind string) {
	fields := strings.Fields(bind)
	_, exists := ModifiersInt[fields[0]]

	if !exists || len(fields[0]) == 1 {
		slog.Error("keybinds", "invalid trigger_label keybind", bind)
	}
}
