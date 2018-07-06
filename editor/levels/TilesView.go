package levels

import (
	"fmt"

	"github.com/inkyblackness/hacked/editor/cmd"
	"github.com/inkyblackness/hacked/editor/event"
	"github.com/inkyblackness/hacked/editor/model"
	"github.com/inkyblackness/hacked/editor/values"
	"github.com/inkyblackness/hacked/ss1/content/archive"
	"github.com/inkyblackness/hacked/ss1/content/archive/level"
	"github.com/inkyblackness/hacked/ss1/content/archive/level/lvlids"
	"github.com/inkyblackness/hacked/ss1/resource"
	"github.com/inkyblackness/hacked/ss1/world/ids"
	"github.com/inkyblackness/hacked/ui/gui"
	"github.com/inkyblackness/imgui-go"
)

// TilesView is for tile properties.
type TilesView struct {
	mod *model.Mod

	guiScale      float32
	commander     cmd.Commander
	eventListener event.Listener

	model tilesViewModel
}

// NewTilesView returns a new instance.
func NewTilesView(mod *model.Mod, guiScale float32, commander cmd.Commander, eventListener event.Listener, eventRegistry event.Registry) *TilesView {
	view := &TilesView{
		mod:           mod,
		guiScale:      guiScale,
		commander:     commander,
		eventListener: eventListener,
		model:         freshTilesViewModel(),
	}
	view.model.selectedTiles.registerAt(eventRegistry)
	return view
}

// WindowOpen returns the flag address, to be used with the main menu.
func (view *TilesView) WindowOpen() *bool {
	return &view.model.windowOpen
}

// Render renders the view.
func (view *TilesView) Render(lvl *level.Level) {
	if view.model.restoreFocus {
		imgui.SetNextWindowFocus()
		view.model.restoreFocus = false
		view.model.windowOpen = true
	}
	if view.model.windowOpen {
		imgui.SetNextWindowSizeV(imgui.Vec2{X: 400 * view.guiScale, Y: 500 * view.guiScale}, imgui.ConditionOnce)
		title := fmt.Sprintf("Level Tiles, %d selected", len(view.model.selectedTiles.list))
		readOnly := !view.editingAllowed(lvl.ID())
		if readOnly {
			title += " (read-only)"
		}
		if imgui.BeginV(title+"###Level Tiles", view.WindowOpen(), 0) {
			view.renderContent(lvl, readOnly)
		}
		imgui.End()
	}
}

func (view *TilesView) renderContent(lvl *level.Level, readOnly bool) {
	isCyberspace := lvl.IsCyberspace()
	tileTypeUnifier := values.NewUnifier()
	floorHeightUnifier := values.NewUnifier()
	ceilingHeightUnifier := values.NewUnifier()
	slopeHeightUnifier := values.NewUnifier()
	slopeControlUnifier := values.NewUnifier()
	musicIndexUnifier := values.NewUnifier()

	floorPaletteIndexUnifier := values.NewUnifier()
	ceilingPaletteIndexUnifier := values.NewUnifier()
	flightPullTypeUnifier := values.NewUnifier()
	gameOfLightStateUnifier := values.NewUnifier()

	floorTextureIndexUnifier := values.NewUnifier()
	floorTextureRotationsUnifier := values.NewUnifier()
	ceilingTextureIndexUnifier := values.NewUnifier()
	ceilingTextureRotationsUnifier := values.NewUnifier()
	wallTextureIndexUnifier := values.NewUnifier()
	wallTextureOffsetUnifier := values.NewUnifier()
	useAdjacentWallTextureUnifier := values.NewUnifier()
	wallTexturePatternUnifier := values.NewUnifier()
	floorLightUnifier := values.NewUnifier()
	ceilingLightUnifier := values.NewUnifier()
	deconstructedUnifier := values.NewUnifier()
	floorHazardUnifier := values.NewUnifier()
	ceilingHazardUnifier := values.NewUnifier()

	multiple := len(view.model.selectedTiles.list) > 0
	for _, pos := range view.model.selectedTiles.list {
		tile := lvl.Tile(int(pos.X.Tile()), int(pos.Y.Tile()))
		tileTypeUnifier.Add(tile.Type)
		floorHeightUnifier.Add(tile.Floor.AbsoluteHeight())
		ceilingHeightUnifier.Add(tile.Ceiling.AbsoluteHeight())
		slopeHeightUnifier.Add(tile.SlopeHeight)
		slopeControlUnifier.Add(tile.Flags.SlopeControl())
		musicIndexUnifier.Add(tile.Flags.MusicIndex())
		if isCyberspace {
			floorPaletteIndexUnifier.Add(tile.TextureInfo.FloorPaletteIndex())
			ceilingPaletteIndexUnifier.Add(tile.TextureInfo.CeilingPaletteIndex())
			flightPullTypeUnifier.Add(tile.Flags.ForCyberspace().FlightPull())
			gameOfLightStateUnifier.Add(tile.Flags.ForCyberspace().GameOfLifeState())
		} else {
			flags := tile.Flags.ForRealWorld()
			floorTextureIndexUnifier.Add(tile.TextureInfo.FloorTextureIndex())
			floorTextureRotationsUnifier.Add(tile.Floor.TextureRotations())
			ceilingTextureIndexUnifier.Add(tile.TextureInfo.CeilingTextureIndex())
			ceilingTextureRotationsUnifier.Add(tile.Ceiling.TextureRotations())
			wallTextureIndexUnifier.Add(tile.TextureInfo.WallTextureIndex())
			wallTextureOffsetUnifier.Add(flags.WallTextureOffset())
			useAdjacentWallTextureUnifier.Add(flags.UseAdjacentWallTexture())
			wallTexturePatternUnifier.Add(flags.WallTexturePattern())
			floorLightUnifier.Add(15 - flags.FloorShadow())
			ceilingLightUnifier.Add(15 - flags.CeilingShadow())
			deconstructedUnifier.Add(flags.Deconstructed())
			floorHazardUnifier.Add(tile.Floor.HasHazard())
			ceilingHazardUnifier.Add(tile.Ceiling.HasHazard())
		}
	}

	imgui.PushItemWidth(-250 * view.guiScale)

	tileTypes := level.TileTypes()
	view.renderCombo(readOnly, multiple, "Tile Type", tileTypeUnifier,
		func(u values.Unifier) int { return int(u.Unified().(level.TileType)) },
		func(value int) string { return tileTypes[value].String() },
		len(tileTypes),
		func(newValue int) {
			view.requestSetTileType(lvl, view.model.selectedTiles.list, level.TileType(newValue))
		})
	view.renderSliderInt(readOnly, multiple, "Floor Height", floorHeightUnifier,
		func(u values.Unifier) int { return int(u.Unified().(level.TileHeightUnit)) },
		func(value int) string { return "%d" },
		0, int(level.TileHeightUnitMax)-1,
		func(newValue int) {
			view.requestSetFloorHeight(lvl, view.model.selectedTiles.list, level.TileHeightUnit(newValue))
		})
	view.renderSliderInt(readOnly, multiple, "Ceiling Height (abs)", ceilingHeightUnifier,
		func(u values.Unifier) int { return int(u.Unified().(level.TileHeightUnit)) },
		func(value int) string { return "%d" },
		0, int(level.TileHeightUnitMax)-1,
		func(newValue int) {
			view.requestSetCeilingHeight(lvl, view.model.selectedTiles.list, level.TileHeightUnit(newValue))
		})
	view.renderSliderInt(readOnly, multiple, "Slope Height", slopeHeightUnifier,
		func(u values.Unifier) int { return int(u.Unified().(level.TileHeightUnit)) },
		func(value int) string { return "%d" },
		0, int(level.TileHeightUnitMax)-1,
		func(newValue int) {
			view.requestSetSlopeHeight(lvl, view.model.selectedTiles.list, level.TileHeightUnit(newValue))
		})
	slopeControls := level.TileSlopeControls()
	view.renderCombo(readOnly, multiple, "Slope Control", slopeControlUnifier,
		func(u values.Unifier) int { return int(u.Unified().(level.TileSlopeControl)) },
		func(value int) string { return slopeControls[value].String() },
		len(slopeControls),
		func(newValue int) {
			view.requestSetSlopeControl(lvl, view.model.selectedTiles.list, slopeControls[newValue])
		})
	view.renderSliderInt(readOnly, multiple, "Music Index", musicIndexUnifier,
		func(u values.Unifier) int { return u.Unified().(int) },
		func(value int) string { return "%d" },
		0, 15,
		func(newValue int) {
			view.requestMusicIndex(lvl, view.model.selectedTiles.list, newValue)
		})

	imgui.Separator()

	if isCyberspace {
		view.renderSliderInt(readOnly, multiple, "Floor Color", floorPaletteIndexUnifier,
			func(u values.Unifier) int { return int(u.Unified().(byte)) },
			func(value int) string { return "%d" },
			0, 255,
			func(newValue int) {
				view.requestFloorPaletteIndex(lvl, view.model.selectedTiles.list, newValue)
			})
		view.renderSliderInt(readOnly, multiple, "Ceiling Color", ceilingPaletteIndexUnifier,
			func(u values.Unifier) int { return int(u.Unified().(byte)) },
			func(value int) string { return "%d" },
			0, 255,
			func(newValue int) {
				view.requestCeilingPaletteIndex(lvl, view.model.selectedTiles.list, newValue)
			})

		flightPulls := level.CyberspaceFlightPulls()
		view.renderCombo(readOnly, multiple, "Flight Pull Type", flightPullTypeUnifier,
			func(u values.Unifier) int { return int(u.Unified().(level.CyberspaceFlightPull)) },
			func(value int) string { return flightPulls[value].String() },
			len(flightPulls),
			func(newValue int) {
				view.requestFlightPullType(lvl, view.model.selectedTiles.list, flightPulls[newValue])
			})
		view.renderSliderInt(readOnly, multiple, "Game Of Life State", gameOfLightStateUnifier,
			func(u values.Unifier) int { return u.Unified().(int) },
			func(value int) string { return "%d" },
			0, 3,
			func(newValue int) {
				view.requestGameOfLightState(lvl, view.model.selectedTiles.list, newValue)
			})

	} else {
		view.renderSliderInt(readOnly, multiple, "Floor Texture", floorTextureIndexUnifier,
			func(u values.Unifier) int { return u.Unified().(int) },
			func(value int) string { return "%d" },
			0, level.FloorCeilingTextureLimit-1,
			func(newValue int) {
				view.requestFloorTextureIndex(lvl, view.model.selectedTiles.list, newValue)
			})
		view.renderSliderInt(readOnly, multiple, "Floor Texture Rotations", floorTextureRotationsUnifier,
			func(u values.Unifier) int { return u.Unified().(int) },
			func(value int) string { return "%d" },
			0, 3,
			func(newValue int) {
				view.requestFloorTextureRotations(lvl, view.model.selectedTiles.list, newValue)
			})

		view.renderSliderInt(readOnly, multiple, "Ceiling Texture", ceilingTextureIndexUnifier,
			func(u values.Unifier) int { return u.Unified().(int) },
			func(value int) string { return "%d" },
			0, level.FloorCeilingTextureLimit-1,
			func(newValue int) {
				view.requestCeilingTextureIndex(lvl, view.model.selectedTiles.list, newValue)
			})
		view.renderSliderInt(readOnly, multiple, "Ceiling Texture Rotations", ceilingTextureRotationsUnifier,
			func(u values.Unifier) int { return u.Unified().(int) },
			func(value int) string { return "%d" },
			0, 3,
			func(newValue int) {
				view.requestCeilingTextureRotations(lvl, view.model.selectedTiles.list, newValue)
			})

		view.renderSliderInt(readOnly, multiple, "Wall Texture", wallTextureIndexUnifier,
			func(u values.Unifier) int { return u.Unified().(int) },
			func(value int) string { return "%d" },
			0, level.DefaultTextureAtlasSize-1,
			func(newValue int) {
				view.requestWallTextureIndex(lvl, view.model.selectedTiles.list, newValue)
			})
		view.renderSliderInt(readOnly, multiple, "Wall Texture Offset", wallTextureOffsetUnifier,
			func(u values.Unifier) int { return int(u.Unified().(level.TileHeightUnit)) },
			func(value int) string { return "%d" },
			0, int(level.TileHeightUnitMax)-1,
			func(newValue int) {
				view.requestWallTextureOffset(lvl, view.model.selectedTiles.list, level.TileHeightUnit(newValue))
			})

		view.renderCheckboxCombo(readOnly, multiple, "Use Adjacent Wall Texture", useAdjacentWallTextureUnifier,
			func(newValue bool) {
				view.requestUseAdjacentWallTexture(lvl, view.model.selectedTiles.list, newValue)
			})
		wallTexturePatterns := level.WallTexturePatterns()
		view.renderCombo(readOnly, multiple, "Wall Texture Pattern", wallTexturePatternUnifier,
			func(u values.Unifier) int { return int(u.Unified().(level.WallTexturePattern)) },
			func(value int) string { return wallTexturePatterns[value].String() },
			len(wallTexturePatterns),
			func(newValue int) {
				view.requestWallTexturePattern(lvl, view.model.selectedTiles.list, wallTexturePatterns[newValue])
			})

		view.renderSliderInt(readOnly, multiple, "Floor Light", floorLightUnifier,
			func(u values.Unifier) int { return u.Unified().(int) },
			func(value int) string { return "%d" },
			0, 15,
			func(newValue int) {
				view.requestFloorLight(lvl, view.model.selectedTiles.list, newValue)
			})
		view.renderSliderInt(readOnly, multiple, "Ceiling Light", ceilingLightUnifier,
			func(u values.Unifier) int { return u.Unified().(int) },
			func(value int) string { return "%d" },
			0, 15,
			func(newValue int) {
				view.requestCeilingLight(lvl, view.model.selectedTiles.list, newValue)
			})

		view.renderCheckboxCombo(readOnly, multiple, "Deconstructed", deconstructedUnifier,
			func(newValue bool) {
				view.requestDeconstructed(lvl, view.model.selectedTiles.list, newValue)
			})
		view.renderCheckboxCombo(readOnly, multiple, "Floor Hazard", floorHazardUnifier,
			func(newValue bool) {
				view.requestFloorHazard(lvl, view.model.selectedTiles.list, newValue)
			})
		view.renderCheckboxCombo(readOnly, multiple, "Ceiling Hazard", ceilingHazardUnifier,
			func(newValue bool) {
				view.requestCeilingHazard(lvl, view.model.selectedTiles.list, newValue)
			})
	}

	imgui.PopItemWidth()
}

func (view *TilesView) renderCheckboxCombo(readOnly, multiple bool, label string, unifier values.Unifier, changeHandler func(bool)) {
	selectedString := ""
	if unifier.IsUnique() {
		selectedValue := unifier.Unified().(bool)
		if selectedValue {
			selectedString = "Yes"
		} else {
			selectedString = "No"
		}
	} else if multiple {
		selectedString = "(multiple)"
	}
	if readOnly {
		imgui.LabelText(label, selectedString)
	} else {
		if imgui.BeginCombo(label, selectedString) {
			for i, text := range []string{"No", "Yes"} {
				if imgui.SelectableV(text, text == selectedString, 0, imgui.Vec2{}) {
					changeHandler(i != 0)
				}
			}
			imgui.EndCombo()
		}
	}
}

func (view *TilesView) renderSliderInt(readOnly, multiple bool, label string, unifier values.Unifier,
	intConverter func(values.Unifier) int, formatter func(int) string, min, max int, changeHandler func(int)) {

	var selectedString string
	selectedValue := -1
	if unifier.IsUnique() {
		selectedValue = intConverter(unifier)
		selectedString = formatter(selectedValue)
	} else if multiple {
		selectedString = "(multiple)"
	}
	if readOnly {
		imgui.LabelText(label, fmt.Sprintf(selectedString, selectedValue))
	} else {
		if gui.StepSliderIntV(label, &selectedValue, min, max, selectedString) {
			changeHandler(selectedValue)
		}
	}
}

func (view *TilesView) renderCombo(readOnly, multiple bool, label string, unifier values.Unifier,
	intConverter func(values.Unifier) int, formatter func(int) string, count int, changeHandler func(int)) {
	var selectedString string
	selectedIndex := -1
	if unifier.IsUnique() {
		selectedIndex = intConverter(unifier)
		selectedString = formatter(selectedIndex)
	} else if multiple {
		selectedString = "(multiple)"
	}
	if readOnly {
		imgui.LabelText(label, selectedString)
	} else {
		if imgui.BeginCombo(label, selectedString) {
			for i := 0; i < count; i++ {
				entryString := formatter(i)

				if imgui.SelectableV(entryString, i == selectedIndex, 0, imgui.Vec2{}) {
					changeHandler(i)
				}
			}
			imgui.EndCombo()
		}
	}
}

func (view *TilesView) editingAllowed(id int) bool {
	gameStateData := view.mod.ModifiedBlocks(resource.LangAny, ids.GameState)
	isSavegame := (len(gameStateData) == 1) && (len(gameStateData[0]) == archive.GameStateSize) && (gameStateData[0][0x009C] > 0)
	moddedLevel := len(view.mod.ModifiedBlocks(resource.LangAny, ids.LevelResourcesStart.Plus(lvlids.PerLevel*id+lvlids.FirstUsed))) > 0

	return moddedLevel && !isSavegame
}

func (view *TilesView) requestSetTileType(lvl *level.Level, positions []MapPosition, tileType level.TileType) {
	view.changeTiles(lvl, positions, func(tile *level.TileMapEntry) {
		tile.Type = tileType
	})
}

func (view *TilesView) requestSetFloorHeight(lvl *level.Level, positions []MapPosition, height level.TileHeightUnit) {
	view.changeTiles(lvl, positions, func(tile *level.TileMapEntry) {
		tile.Floor = tile.Floor.WithAbsoluteHeight(height)
	})
}

func (view *TilesView) requestSetCeilingHeight(lvl *level.Level, positions []MapPosition, height level.TileHeightUnit) {
	view.changeTiles(lvl, positions, func(tile *level.TileMapEntry) {
		tile.Ceiling = tile.Ceiling.WithAbsoluteHeight(height)
	})
}

func (view *TilesView) requestSetSlopeHeight(lvl *level.Level, positions []MapPosition, height level.TileHeightUnit) {
	view.changeTiles(lvl, positions, func(tile *level.TileMapEntry) {
		tile.SlopeHeight = height
	})
}

func (view *TilesView) requestSetSlopeControl(lvl *level.Level, positions []MapPosition, value level.TileSlopeControl) {
	view.changeTiles(lvl, positions, func(tile *level.TileMapEntry) {
		tile.Flags = tile.Flags.WithSlopeControl(value)
	})
}

func (view *TilesView) requestMusicIndex(lvl *level.Level, positions []MapPosition, value int) {
	view.changeTiles(lvl, positions, func(tile *level.TileMapEntry) {
		tile.Flags = tile.Flags.WithMusicIndex(value)
	})
}

func (view *TilesView) requestFloorTextureIndex(lvl *level.Level, positions []MapPosition, value int) {
	view.changeTiles(lvl, positions, func(tile *level.TileMapEntry) {
		tile.TextureInfo = tile.TextureInfo.WithFloorTextureIndex(value)
	})
}

func (view *TilesView) requestFloorTextureRotations(lvl *level.Level, positions []MapPosition, value int) {
	view.changeTiles(lvl, positions, func(tile *level.TileMapEntry) {
		tile.Floor = tile.Floor.WithTextureRotations(value)
	})
}

func (view *TilesView) requestCeilingTextureIndex(lvl *level.Level, positions []MapPosition, value int) {
	view.changeTiles(lvl, positions, func(tile *level.TileMapEntry) {
		tile.TextureInfo = tile.TextureInfo.WithCeilingTextureIndex(value)
	})
}

func (view *TilesView) requestCeilingTextureRotations(lvl *level.Level, positions []MapPosition, value int) {
	view.changeTiles(lvl, positions, func(tile *level.TileMapEntry) {
		tile.Ceiling = tile.Ceiling.WithTextureRotations(value)
	})
}

func (view *TilesView) requestWallTextureIndex(lvl *level.Level, positions []MapPosition, value int) {
	view.changeTiles(lvl, positions, func(tile *level.TileMapEntry) {
		tile.TextureInfo = tile.TextureInfo.WithWallTextureIndex(value)
	})
}

func (view *TilesView) requestWallTextureOffset(lvl *level.Level, positions []MapPosition, value level.TileHeightUnit) {
	view.changeTiles(lvl, positions, func(tile *level.TileMapEntry) {
		tile.Flags = tile.Flags.ForRealWorld().WithWallTextureOffset(value).AsTileFlag()
	})
}

func (view *TilesView) requestUseAdjacentWallTexture(lvl *level.Level, positions []MapPosition, value bool) {
	view.changeTiles(lvl, positions, func(tile *level.TileMapEntry) {
		tile.Flags = tile.Flags.ForRealWorld().WithUseAdjacentWallTexture(value).AsTileFlag()
	})
}

func (view *TilesView) requestWallTexturePattern(lvl *level.Level, positions []MapPosition, value level.WallTexturePattern) {
	view.changeTiles(lvl, positions, func(tile *level.TileMapEntry) {
		tile.Flags = tile.Flags.ForRealWorld().WithWallTexturePattern(value).AsTileFlag()
	})
}

func (view *TilesView) requestFloorLight(lvl *level.Level, positions []MapPosition, value int) {
	view.changeTiles(lvl, positions, func(tile *level.TileMapEntry) {
		tile.Flags = tile.Flags.ForRealWorld().WithFloorShadow(15 - value).AsTileFlag()
	})
}

func (view *TilesView) requestCeilingLight(lvl *level.Level, positions []MapPosition, value int) {
	view.changeTiles(lvl, positions, func(tile *level.TileMapEntry) {
		tile.Flags = tile.Flags.ForRealWorld().WithCeilingShadow(15 - value).AsTileFlag()
	})
}

func (view *TilesView) requestDeconstructed(lvl *level.Level, positions []MapPosition, value bool) {
	view.changeTiles(lvl, positions, func(tile *level.TileMapEntry) {
		tile.Flags = tile.Flags.ForRealWorld().WithDeconstructed(value).AsTileFlag()
	})
}

func (view *TilesView) requestFloorHazard(lvl *level.Level, positions []MapPosition, value bool) {
	view.changeTiles(lvl, positions, func(tile *level.TileMapEntry) {
		tile.Floor = tile.Floor.WithHazard(value)
	})
}

func (view *TilesView) requestCeilingHazard(lvl *level.Level, positions []MapPosition, value bool) {
	view.changeTiles(lvl, positions, func(tile *level.TileMapEntry) {
		tile.Ceiling = tile.Ceiling.WithHazard(value)
	})
}

func (view *TilesView) requestFloorPaletteIndex(lvl *level.Level, positions []MapPosition, value int) {
	view.changeTiles(lvl, positions, func(tile *level.TileMapEntry) {
		tile.TextureInfo = tile.TextureInfo.WithFloorPaletteIndex(byte(value))
	})
}

func (view *TilesView) requestCeilingPaletteIndex(lvl *level.Level, positions []MapPosition, value int) {
	view.changeTiles(lvl, positions, func(tile *level.TileMapEntry) {
		tile.TextureInfo = tile.TextureInfo.WithCeilingPaletteIndex(byte(value))
	})
}

func (view *TilesView) requestFlightPullType(lvl *level.Level, positions []MapPosition, value level.CyberspaceFlightPull) {
	view.changeTiles(lvl, positions, func(tile *level.TileMapEntry) {
		tile.Flags = tile.Flags.ForCyberspace().WithFlightPull(value).AsTileFlag()
	})
}

func (view *TilesView) requestGameOfLightState(lvl *level.Level, positions []MapPosition, value int) {
	view.changeTiles(lvl, positions, func(tile *level.TileMapEntry) {
		tile.Flags = tile.Flags.ForCyberspace().WithGameOfLifeState(value).AsTileFlag()
	})
}

func (view *TilesView) changeTiles(lvl *level.Level, positions []MapPosition, modifier func(*level.TileMapEntry)) {
	for _, pos := range positions {
		tile := lvl.Tile(int(pos.X.Tile()), int(pos.Y.Tile()))
		modifier(tile)
	}

	command := patchLevelDataCommand{
		restoreState: func() {
			view.model.restoreFocus = true
			view.setSelectedTiles(positions)
			view.setSelectedLevel(lvl.ID())
		},
	}

	newDataSet := lvl.EncodeState()
	for id, newData := range newDataSet {
		if len(newData) > 0 {
			resourceID := ids.LevelResourcesStart.Plus(lvlids.PerLevel*lvl.ID() + id)
			patch, changed, err := view.mod.CreateBlockPatch(resource.LangAny, resourceID, 0, newData)
			if err != nil {
				fmt.Printf("err: %v\n", err)
				// TODO how to handle this? We're not expecting this, so crash and burn?
			} else if changed {
				command.patches = append(command.patches, patch)
			}
		}
	}

	view.commander.Queue(command)
}

func (view *TilesView) setSelectedLevel(id int) {
	view.eventListener.Event(LevelSelectionSetEvent{id: id})
}

func (view *TilesView) setSelectedTiles(positions []MapPosition) {
	view.eventListener.Event(TileSelectionSetEvent{tiles: positions})
}