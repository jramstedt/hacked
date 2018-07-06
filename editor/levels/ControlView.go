package levels

import (
	"reflect"

	"github.com/inkyblackness/hacked/editor/event"
	"github.com/inkyblackness/hacked/ss1/content/archive"
	"github.com/inkyblackness/hacked/ui/gui"
	"github.com/inkyblackness/imgui-go"
)

// ControlView is the core view for level editing.
type ControlView struct {
	guiScale float32

	eventListener event.Listener

	model controlViewModel
}

// NewControlView returns a new instance.
func NewControlView(guiScale float32, eventListener event.Listener, eventRegistry event.Registry) *ControlView {
	view := &ControlView{
		guiScale:      guiScale,
		eventListener: eventListener,
		model:         freshControlViewModel(),
	}
	var evt LevelSelectionSetEvent
	eventRegistry.RegisterHandler(reflect.TypeOf(evt), view.onLevelSelectionSetEvent)
	return view
}

// WindowOpen returns the flag address, to be used with the main menu.
func (view *ControlView) WindowOpen() *bool {
	return &view.model.windowOpen
}

// SelectedLevel returns the currently selected level.
func (view *ControlView) SelectedLevel() int {
	return view.model.selectedLevel
}

// Render renders the view.
func (view *ControlView) Render() {
	if view.model.restoreFocus {
		imgui.SetNextWindowFocus()
		view.model.restoreFocus = false
		view.model.windowOpen = true
	}
	if view.model.windowOpen {
		imgui.SetNextWindowSizeV(imgui.Vec2{X: 400 * view.guiScale, Y: 300 * view.guiScale}, imgui.ConditionOnce)
		if imgui.BeginV("Level Control", view.WindowOpen(), imgui.WindowFlagsNoCollapse) {
			view.renderContent()
		}
		imgui.End()
	}
}

func (view *ControlView) renderContent() {
	imgui.PushItemWidth(-100 * view.guiScale)
	gui.StepSliderInt("Active Level", &view.model.selectedLevel, 0, archive.MaxLevels-1)
	imgui.PopItemWidth()
}

func (view *ControlView) onLevelSelectionSetEvent(evt LevelSelectionSetEvent) {
	view.model.selectedLevel = evt.id
}