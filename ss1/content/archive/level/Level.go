package level

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"

	"github.com/inkyblackness/hacked/ss1/content/archive/level/lvlids"
	"github.com/inkyblackness/hacked/ss1/resource"
	"github.com/inkyblackness/hacked/ss1/serial"
)

// Level is the complete structure defining all necessary data for a level.
type Level struct {
	id        int
	localizer resource.Localizer

	resStart resource.ID
	resEnd   resource.ID

	baseInfo       BaseInfo
	tileMap        TileMap
	wallHeightsMap WallHeightsMap
}

// NewLevel returns a new instance.
func NewLevel(resourceBase resource.ID, id int, localizer resource.Localizer) *Level {
	lvl := &Level{
		id: id,

		localizer: localizer,

		resStart:       resourceBase.Plus(lvlids.PerLevel * id),
		tileMap:        NewTileMap(64, 64),
		wallHeightsMap: NewWallHeightsMap(64, 64),
	}
	lvl.resEnd = lvl.resStart.Plus(lvlids.PerLevel)

	lvl.reloadBaseInfo()
	lvl.reloadTileMap()

	return lvl
}

// ID returns the identifier of the level.
func (lvl Level) ID() int {
	return lvl.id
}

// InvalidateResources resets all internally cached data.
func (lvl *Level) InvalidateResources(ids []resource.ID) {
	for _, id := range ids {
		if (id >= lvl.resStart) && (id < lvl.resEnd) {
			lvl.onLevelResourceDataChanged(int(id.Value() - lvl.resStart.Value()))
		}
	}
}

// Size returns the dimensions of the level.
func (lvl Level) Size() (x, y int, z HeightShift) {
	return int(lvl.baseInfo.XSize), int(lvl.baseInfo.YSize), lvl.baseInfo.ZShift
}

// IsCyberspace returns true if the level describes a cyberspace.
func (lvl Level) IsCyberspace() bool {
	return lvl.baseInfo.Cyberspace != 0
}

// Tile returns the tile entry at given position.
func (lvl *Level) Tile(x, y int) *TileMapEntry {
	return lvl.tileMap.Tile(x, y)
}

// MapGridInfo returns the information necessary to draw a 2D map.
func (lvl *Level) MapGridInfo(x, y int) (TileType, TileSlopeControl, WallHeights) {
	tile := lvl.tileMap.Tile(x, y)
	if tile == nil {
		return TileTypeSolid, TileSlopeControlCeilingInverted, WallHeights{}
	}
	return tile.Type, tile.Flags.SlopeControl(), *lvl.wallHeightsMap.Tile(x, y)
}

// EncodeState returns a subset of encoded level data, which only includes
// data that is loaded (modified) by the level structure.
// For any data block that is not relevant, a zero length slice is returned.
func (lvl *Level) EncodeState() [lvlids.PerLevel][]byte {
	var levelData [lvlids.PerLevel][]byte

	levelData[lvlids.Information] = encode(&lvl.baseInfo)

	levelData[lvlids.TileMap] = encode(lvl.tileMap)

	return levelData
}

func (lvl Level) encode(data interface{}) []byte {
	buf := bytes.NewBuffer(nil)
	encoder := serial.NewEncoder(buf)
	encoder.Code(data)
	return buf.Bytes()
}

func (lvl *Level) onLevelResourceDataChanged(id int) {
	switch id {
	case lvlids.Information:
		lvl.reloadBaseInfo()
	case lvlids.TileMap:
		lvl.reloadTileMap()
	}
}

func (lvl *Level) reloadBaseInfo() {
	reader, err := lvl.reader(lvlids.Information)
	if err != nil {
		lvl.baseInfo = BaseInfo{}
		return
	}
	err = binary.Read(reader, binary.LittleEndian, &lvl.baseInfo)
	if err != nil {
		lvl.baseInfo = BaseInfo{}
	}
}

func (lvl *Level) reloadTileMap() {
	reader, err := lvl.reader(lvlids.TileMap)
	if err == nil {
		coder := serial.NewDecoder(reader)
		lvl.tileMap.Code(coder)
		err = coder.FirstError()
	}
	if err != nil {
		lvl.clearTileMap()
	}
	lvl.wallHeightsMap.CalculateFrom(lvl.tileMap)
}

func (lvl *Level) clearTileMap() {
	for _, row := range lvl.tileMap {
		for i := 0; i < len(row); i++ {
			row[i].Reset()
		}
	}
}

func (lvl *Level) reader(block int) (io.Reader, error) {
	res, err := lvl.localizer.LocalizedResources(resource.LangAny).Select(lvl.resStart.Plus(block))
	if err != nil {
		return nil, err
	}
	if res.ContentType() != resource.Archive {
		return nil, errors.New("resource is not for archive")
	}
	if res.BlockCount() != 1 {
		return nil, errors.New("resource has invalid block count")
	}
	return res.Block(0)
}