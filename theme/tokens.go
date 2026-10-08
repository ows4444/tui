package theme

import "github.com/ows4444/tui/ansi"

// Tokens are one component's colour overrides. A nil field leaves the theme's
// role in place, so a Tokens value names only what differs.
type Tokens struct {
	Text       ansi.Color // Theme.Text
	Muted      ansi.Color // Theme.Muted
	Accent     ansi.Color // Theme.Primary
	Focus      ansi.Color // Theme.Focus
	Selection  ansi.Color // Theme.Selection
	Border     ansi.Color // Theme.BorderColor
	Background ansi.Color // Theme.Background
	Surface    ansi.Color // Theme.Surface

	// Semantic roles. They follow the same nil-means-inherit rule.
	Success     ansi.Color // Theme.Success
	Warning     ansi.Color // Theme.Warning
	Error       ansi.Color // Theme.Error
	Info        ansi.Color // Theme.Info
	Overlay     ansi.Color // Theme.Overlay
	TextInverse ansi.Color // Theme.TextInverse
}

// Component names accepted by WithTokens, TokensFor, ForComponent and Resolve.
// Use these constants rather than string literals: a misspelt name overrides
// nothing, silently. Components lists them all.
const (
	ComponentAccordion          = "accordion"
	ComponentAppShell           = "appshell"
	ComponentAutocomplete       = "autocomplete"
	ComponentButton             = "button"
	ComponentColorPicker        = "colorpicker"
	ComponentCommandPalette     = "commandpalette"
	ComponentConfirm            = "confirm"
	ComponentContextMenu        = "contextmenu"
	ComponentDataTable          = "datatable"
	ComponentDatePicker         = "datepicker"
	ComponentDialog             = "dialog"
	ComponentDrawer             = "drawer"
	ComponentErrorRetry         = "errorretry"
	ComponentFaces              = "faces"
	ComponentFilePicker         = "filepicker"
	ComponentForm               = "form"
	ComponentHelpScreen         = "helpscreen"
	ComponentImageView          = "imageview"
	ComponentLoadingBar         = "loadingbar"
	ComponentMarkdown           = "markdown"
	ComponentMaskedInput        = "maskedinput"
	ComponentMenu               = "menu"
	ComponentMenuBar            = "menubar"
	ComponentMultiSelect        = "multiselect"
	ComponentNotificationCenter = "notificationcenter"
	ComponentPasswordInput      = "passwordinput"
	ComponentPicker             = "picker"
	ComponentPopover            = "popover"
	ComponentRadioGroup         = "radiogroup"
	ComponentRating             = "rating"
	ComponentScrollbar          = "scrollbar"
	ComponentSkeleton           = "skeleton"
	ComponentSlider             = "slider"
	ComponentSpinner            = "spinner"
	ComponentSplitPane          = "splitpane"
	ComponentStreamText         = "streamtext"
	ComponentTabs               = "tabs"
	ComponentTagInput           = "taginput"
	ComponentTextArea           = "textarea"
	ComponentTextInput          = "textinput"
	ComponentToast              = "toast"
	ComponentToolApproval       = "toolapproval"
	ComponentTreeView           = "treeview"
	ComponentWizard             = "wizard"
)

// Components returns every component name above, sorted.
func Components() []string {
	return []string{
		ComponentAccordion, ComponentAppShell, ComponentAutocomplete,
		ComponentButton, ComponentColorPicker, ComponentCommandPalette,
		ComponentConfirm, ComponentContextMenu, ComponentDataTable,
		ComponentDatePicker, ComponentDialog, ComponentDrawer,
		ComponentErrorRetry, ComponentFaces, ComponentFilePicker,
		ComponentForm, ComponentHelpScreen, ComponentImageView,
		ComponentLoadingBar, ComponentMarkdown, ComponentMaskedInput, ComponentMenu,
		ComponentMenuBar, ComponentMultiSelect, ComponentNotificationCenter,
		ComponentPasswordInput, ComponentPicker, ComponentPopover,
		ComponentRadioGroup, ComponentRating,
		ComponentScrollbar, ComponentSkeleton, ComponentSlider, ComponentSpinner,
		ComponentSplitPane, ComponentStreamText, ComponentTabs,
		ComponentTagInput, ComponentTextArea, ComponentTextInput,
		ComponentToast, ComponentToolApproval, ComponentTreeView,
		ComponentWizard,
	}
}

// ComponentTokens holds per-component Tokens. It is immutable once built, so
// Theme stays comparable and copies of a Theme can share it.
type ComponentTokens struct{ m map[string]Tokens }

// WithTokens returns a copy of t whose component (a widget name such as
// "tabs" or "toast") carries tok. t is not modified and the copy shares no
// state with it. A second call for the same component replaces the first.
func (t Theme) WithTokens(component string, tok Tokens) Theme {
	m := map[string]Tokens{component: tok}
	if t.Components != nil {
		for k, v := range t.Components.m {
			if k != component {
				m[k] = v
			}
		}
	}
	t.Components = &ComponentTokens{m: m}
	return t
}

// TokensFor returns the tokens of component, every field resolved from the
// theme's roles and then overridden by anything set with WithTokens.
func (t Theme) TokensFor(component string) Tokens {
	tok := Tokens{
		Text: t.Text, Muted: t.Muted, Accent: t.Primary, Focus: t.Focus,
		Selection: t.Selection, Border: t.BorderColor,
		Background: t.background(), Surface: t.Surface,
		Success: t.Success, Warning: t.Warning, Error: t.Error, Info: t.Info,
		Overlay: t.Overlay, TextInverse: t.TextInverse,
	}
	var o Tokens
	if t.Components != nil {
		o = t.Components.m[component]
	}
	return tok.Over(o)
}

// fields returns pointers to every field of tok, in declaration order.
func (tok *Tokens) fields() []*ansi.Color {
	return []*ansi.Color{
		&tok.Text, &tok.Muted, &tok.Accent, &tok.Focus, &tok.Selection,
		&tok.Border, &tok.Background, &tok.Surface, &tok.Success, &tok.Warning,
		&tok.Error, &tok.Info, &tok.Overlay, &tok.TextInverse,
	}
}

// Over returns tok with every non-nil field of o laid on top.
func (tok Tokens) Over(o Tokens) Tokens {
	dst, src := tok.fields(), o.fields()
	for i, p := range src {
		if *p != nil {
			*dst[i] = *p
		}
	}
	return tok
}

// IsZero reports whether tok overrides nothing.
func (tok Tokens) IsZero() bool {
	for _, p := range tok.fields() {
		if *p != nil {
			return false
		}
	}
	return true
}

// apply writes the non-nil fields of o onto the matching roles of t.
//
// States the theme derived from its roles (every preset fills them) follow the
// new roles; a state style the theme set to something else is kept.
func (t Theme) apply(o Tokens) Theme {
	if o.IsZero() {
		return t
	}
	before := t.derivedStates()
	t.Text = pick(t.Text, o.Text)
	t.Muted = pick(t.Muted, o.Muted)
	t.Primary = pick(t.Primary, o.Accent)
	t.Focus = pick(t.Focus, o.Focus)
	t.Selection = pick(t.Selection, o.Selection)
	t.BorderColor = pick(t.BorderColor, o.Border)
	t.Background = pick(t.Background, o.Background)
	t.Surface = pick(t.Surface, o.Surface)
	t.Success = pick(t.Success, o.Success)
	t.Warning = pick(t.Warning, o.Warning)
	t.Error = pick(t.Error, o.Error)
	t.Info = pick(t.Info, o.Info)
	t.Overlay = pick(t.Overlay, o.Overlay)
	t.TextInverse = pick(t.TextInverse, o.TextInverse)
	after := t.derivedStates()
	if t.States.Focus == before.Focus {
		t.States.Focus = after.Focus
	}
	if t.States.Hover == before.Hover {
		t.States.Hover = after.Hover
	}
	if t.States.Disabled == before.Disabled {
		t.States.Disabled = after.Disabled
	}
	if t.States.Selected == before.Selected {
		t.States.Selected = after.Selected
	}
	return t
}

// derivedStates is what ResolvedStates yields when the theme set no States.
func (t Theme) derivedStates() States {
	t.States = States{}
	return t.ResolvedStates()
}

func pick(base, over ansi.Color) ansi.Color {
	if over != nil {
		return over
	}
	return base
}

// Resolve returns t with the overrides registered for component (WithTokens)
// applied, then inst, a per-instance override, on top. Only overridden roles
// change; with nothing to apply it returns t unchanged. Widgets call it to
// build the theme they render with.
func (t Theme) Resolve(component string, inst Tokens) Theme {
	if t.Components != nil {
		if o, ok := t.Components.m[component]; ok {
			t = t.apply(o)
		}
	}
	return t.apply(inst)
}

// ForComponent returns t with component's tokens applied to its colour roles.
// Give the result to that widget's SetTheme (or Theme field) and only that
// widget changes; t, and every other widget, keeps the shared theme.
func (t Theme) ForComponent(component string) Theme {
	return t.apply(t.TokensFor(component))
}
