package ui

import (
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

func parseAiKeybinds() {
	aibinds = make(util.Keybinds)

	for _, v := range config.Cfg.Keys.Ai.ClearSession {
		aibinds.Validate(v)
		aibinds.Bind(v, aiClearSession)
	}

	for _, v := range config.Cfg.Keys.Ai.CopyLastResponse {
		aibinds.Validate(v)
		aibinds.Bind(v, aiCopyLast)
	}

	for _, v := range config.Cfg.Keys.Ai.ResumeSession {
		aibinds.Validate(v)
		aibinds.Bind(v, aiResume)
	}

	for _, v := range config.Cfg.Keys.Ai.RunLastResponse {
		aibinds.Validate(v)
		aibinds.Bind(v, aiExecuteLast)
	}
}

func parseKeybinds() {
	binds = make(util.Keybinds)

	for _, v := range config.Cfg.Keys.AcceptTypeahead {
		if ok := binds.Validate(v); ok {
			binds.Bind(v, acceptTypeahead)
		}
	}

	for _, v := range config.Cfg.Keys.Close {
		binds.Validate(v)
		binds.Bind(v, quitKeybind)
	}

	for _, v := range config.Cfg.Keys.Next {
		binds.Validate(v)
		binds.Bind(v, selectNext)
	}

	for _, v := range config.Cfg.Keys.Prev {
		binds.Validate(v)
		binds.Bind(v, selectPrev)
	}

	for _, v := range config.Cfg.Keys.RemoveFromHistory {
		binds.Validate(v)
		binds.Bind(v, deleteFromHistory)
	}

	for _, v := range config.Cfg.Keys.ResumeQuery {
		binds.Validate(v)
		binds.Bind(v, resume)
	}

	for _, v := range config.Cfg.Keys.ToggleExactSearch {
		binds.Validate(v)
		binds.Bind(v, toggleExactMatch)
	}

	binds.Bind("enter", func() bool { return activate(false, false) })
	binds.Bind(strings.Join([]string{config.Cfg.Keys.ActivationModifiers.KeepOpen, "enter"}, " "), func() bool { return activate(true, false) })
	binds.Bind(strings.Join([]string{config.Cfg.Keys.ActivationModifiers.Alternate, "enter"}, " "), func() bool { return activate(false, true) })

	keepOpenModifier = util.Modifiers[config.Cfg.Keys.ActivationModifiers.KeepOpen]
	activateAltModifier = util.Modifiers[config.Cfg.Keys.ActivationModifiers.Alternate]

	binds.ValidateTriggerLabels(config.Cfg.Keys.TriggerLabels)
	labelTrigger = util.ModifiersInt[strings.Fields(config.Cfg.Keys.TriggerLabels)[0]]
	labelModifier = util.Modifiers[strings.Fields(config.Cfg.Keys.TriggerLabels)[0]]
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
