package ui

import (
	"log/slog"
	"slices"
	"strings"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/modules"
	"github.com/abenz1267/walker/internal/modules/clipboard"
	"github.com/abenz1267/walker/internal/util"
	"github.com/diamondburned/gotk4/pkg/core/gioutil"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
)

var (
	binds   util.Keybinds
	aibinds util.Keybinds

	labelTrigger        = gdk.KEY_Alt_L
	keepOpenModifier    = gdk.ShiftMask
	labelModifier       = gdk.AltMask
	activateAltModifier = gdk.AltMask
)

type Keybinds map[string]util.Keybinds

var (
	keybinds       Keybinds
	builtinActions = map[string]func(){}
)

func setupBuiltinActions() {
	builtinActions = make(map[string]func())
	builtinActions["%ACTIVATE%"] = func() { activate(false, false) }
	builtinActions["%ACTIVATE_KEEP_OPEN%"] = func() { activate(true, false) }
	builtinActions["%TOGGLE_LABELS%"] = func() { toggleAM() }
	builtinActions["%ACCEPT_TYPEAHEAD%"] = func() { elements.input.GrabFocus() }
	builtinActions["%NEXT%"] = func() { selectNext() }
	builtinActions["%PREV%"] = func() { selectPrev() }
	builtinActions["%CLOSE%"] = func() { quitKeybind() }
	builtinActions["%REMOVE_FROM_HISTORY%"] = func() { deleteFromHistory() }
	builtinActions["%RESUME_QUERY%"] = func() { resume() }
	builtinActions["%TOGGLE_EXACT_SEARCH%"] = func() { toggleExactMatch() }
}

func parseKeybinds() {
	keybinds = make(Keybinds)
	keybinds["global"] = make(util.Keybinds)

	for _, v := range config.Cfg.Keybinds {
		if ok := util.ValidateKeybind(v.Key); ok {
			key, modifier := util.ParseKeybind(v.Key)

			if _, ok := keybinds["global"][key]; !ok {
				keybinds["global"][key] = make(map[gdk.ModifierType]util.KeybindCommand)
			}

			if _, ok := keybinds["global"][key][modifier]; !ok {
				keybinds["global"][key][modifier] = v
			} else {
				slog.Error("keybinds", "duplicate keybind", v.Key, "module", "global")
			}
		}
	}
}

func toggleAM() bool {
	if config.Cfg.ActivationMode.Disabled {
		return false
	}

	if common.selection.NItems() != 0 {
		enableAM()

		return true
	}

	return false
}

func deleteFromHistory() bool {
	entry := gioutil.ObjectValue[util.Entry](common.items.Item(common.selection.Selected()))
	deleted := hstry.Delete(entry.Identifier())

	if !deleted && singleModule != nil && singleModule.General().Name == config.Cfg.Builtins.Clipboard.Name {
		entry := gioutil.ObjectValue[util.Entry](common.items.Item(common.selection.Selected()))
		singleModule.(*clipboard.Clipboard).Delete(entry)
		debouncedProcess(process)
		return true
	}

	debouncedProcess(process)

	return true
}

func aiCopyLast() bool {
	if !isAi {
		return false
	}

	ai := findModule(config.Cfg.Builtins.AI.Name, toUse, explicits).(*modules.AI)
	ai.CopyLastResponse()

	return true
}

func aiExecuteLast() bool {
	if !isAi {
		return false
	}

	ai := findModule(config.Cfg.Builtins.AI.Name, toUse, explicits).(*modules.AI)
	ai.RunLastMessageInTerminal()
	quit(true)

	return true
}

func toggleExactMatch() bool {
	text := elements.input.Text()

	if strings.HasPrefix(text, "'") {
		elements.input.SetText(strings.TrimPrefix(text, "'"))
	} else {
		elements.input.SetText("'" + text)
	}

	elements.input.SetPosition(-1)

	return true
}

func resume() bool {
	if appstate.LastQuery != "" {
		elements.input.SetText(appstate.LastQuery)
		elements.input.SetPosition(-1)
		elements.input.GrabFocus()
	}

	return true
}

func aiResume() bool {
	if !isAi {
		return false
	}

	ai := findModule(config.Cfg.Builtins.AI.Name, toUse, explicits).(*modules.AI)
	ai.ResumeLastMessages()

	return true
}

func aiClearSession() bool {
	if !isAi {
		return false
	}

	ai := findModule(config.Cfg.Builtins.AI.Name, toUse, explicits).(*modules.AI)
	elements.input.SetText("")
	ai.ClearCurrent()

	return true
}

func activateFunctionKeys(val uint) bool {
	index := slices.Index(fkeys, val)

	if index != -1 {
		selectActivationMode(false, true, uint(index))
		return true
	}

	return false
}

func activateKeepOpenFunctionKeys(val uint) bool {
	index := slices.Index(fkeys, val)

	if index != -1 {
		selectActivationMode(true, true, uint(index))
		return true
	}

	return false
}

func quitKeybind() bool {
	if appstate.IsDmenu {
		handleDmenuResult("CNCLD")
	}

	if config.Cfg.IsService {
		quit(false)
		return true
	} else {
		exit(false, true)
		return true
	}
}

func acceptTypeahead() bool {
	if elements.typeahead.Text() != "" {
		tahAcceptedIdentifier = tahSuggestionIdentifier
		tahSuggestionIdentifier = ""

		elements.input.SetText(elements.typeahead.Text())
		elements.input.SetPosition(-1)

		return true
	}

	return false
}

func activate(keepOpen bool, isAlt bool) bool {
	if appstate.ForcePrint && elements.grid.Model().NItems() == 0 {
		if appstate.IsDmenu {
			handleDmenuResult(elements.input.Text())
		}

		closeAfterActivation(keepOpen, false)
		return true
	}

	activateItem(keepOpen, isAlt)
	return true
}
