package movie

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io/ioutil"

	"github.com/inkyblackness/hacked/ss1/content/audio"
	"github.com/inkyblackness/hacked/ss1/content/bitmap"
	"github.com/inkyblackness/hacked/ss1/content/movie/internal/compression"
	"github.com/inkyblackness/hacked/ss1/content/text"
	"github.com/inkyblackness/hacked/ss1/resource"
	"github.com/inkyblackness/hacked/ss1/serial"
	"github.com/inkyblackness/hacked/ss1/serial/rle"
)

// Cache retrieves movie container from a localizer and keeps them decoded until they are invalidated.
type Cache struct {
	cp text.Codepage

	localizer resource.Localizer

	movies map[resource.Key]*cachedMovie
}

type cachedMovie struct {
	cp text.Codepage

	container Container

	sound           *audio.L8
	scenes          []Scene
	subtitlesByLang map[resource.Language]*Subtitles
}

func (cached *cachedMovie) audio() audio.L8 {
	if cached.sound != nil {
		return *cached.sound
	}
	var samples []byte
	for _, entry := range cached.container.Entries {
		if entry.Type() == Audio {
			samples = append(samples, entry.Data()...)
		}
	}
	cached.sound = &audio.L8{
		Samples:    samples,
		SampleRate: float32(cached.container.AudioSampleRate),
	}
	return *cached.sound
}

func (cached *cachedMovie) video() []Scene {
	if len(cached.scenes) > 0 {
		return cached.scenes
	}

	var scenes []Scene
	var currentPalette bitmap.Palette
	width := int(cached.container.VideoWidth)
	height := int(cached.container.VideoHeight)
	frameBuffer := make([]byte, width*height)
	decoderBuilder := compression.NewFrameDecoderBuilder(width, height)
	decoderBuilder.ForStandardFrame(frameBuffer, width)

	clonePalette := func() *bitmap.Palette {
		paletteCopy := currentPalette
		return &paletteCopy
	}
	cloneFramebuffer := func() []byte {
		bufferCopy := make([]byte, len(frameBuffer))
		copy(bufferCopy, frameBuffer)
		return bufferCopy
	}

	var currentScene *Scene
	for _, entry := range cached.container.Entries {
		switch entry.Type() {
		case Palette:
			{
				if currentScene != nil {
					scenes = append(scenes, *currentScene)
				}
				currentScene = nil
				decoder := serial.NewDecoder(bytes.NewReader(entry.Data()))
				decoder.Code(&currentPalette)
			}
		case ControlDictionary:
			{
				words, wordsErr := compression.UnpackControlWords(entry.Data())

				if wordsErr == nil {
					decoderBuilder.WithControlWords(words)
				}
			}
		case PaletteLookupList:
			{
				if currentScene != nil {
					scenes = append(scenes, *currentScene)
				}
				currentScene = nil
				decoderBuilder.WithPaletteLookupList(entry.Data())
			}
		case LowResVideo:
			{
				var videoHeader LowResVideoHeader
				reader := bytes.NewReader(entry.Data())

				err := binary.Read(reader, binary.LittleEndian, &videoHeader)
				if err != nil {
					break
				}
				frameErr := rle.Decompress(reader, frameBuffer)
				if frameErr == nil {
					// TODO
				}
			}
		case HighResVideo:
			{
				var videoHeader HighResVideoHeader
				reader := bytes.NewReader(entry.Data())

				err := binary.Read(reader, binary.LittleEndian, &videoHeader)
				if err != nil {
					break
				}
				bitstreamData := entry.Data()[HighResVideoHeaderSize:videoHeader.PixelDataOffset]
				maskstreamData := entry.Data()[videoHeader.PixelDataOffset:]
				decoder := decoderBuilder.Build()

				err = decoder.Decode(bitstreamData, maskstreamData)
				if err != nil {
					break
				}
				if currentScene == nil {
					currentScene = &Scene{}
				}

				bmp := bitmap.Bitmap{
					Header: bitmap.Header{
						Type:   bitmap.TypeFlat8Bit,
						Width:  int16(cached.container.VideoWidth),
						Height: int16(cached.container.VideoHeight),
						Stride: cached.container.VideoWidth,
					},
					Palette: clonePalette(),
					Pixels:  cloneFramebuffer(),
				}
				currentScene.Frames = append(currentScene.Frames, bmp)
			}
		}
	}

	cached.scenes = scenes

	return cached.scenes
}

func (cached *cachedMovie) subtitles(language resource.Language) Subtitles {
	sub := cached.subtitlesByLang[language]
	if sub != nil {
		return *sub
	}

	sub = &Subtitles{}
	expectedControl := SubtitleControlForLanguage(language)

	for _, entry := range cached.container.Entries {
		if entry.Type() != Subtitle {
			continue
		}
		var subtitleHeader SubtitleHeader
		err := binary.Read(bytes.NewReader(entry.Data()), binary.LittleEndian, &subtitleHeader)
		if err != nil {
			continue
		}
		if subtitleHeader.Control == expectedControl {
			sub.add(entry.Timestamp(), cached.cp.Decode(entry.Data()[subtitleHeader.TextOffset:]))
		}
	}
	if (len(sub.Entries) > 0) && (len(sub.Entries[len(sub.Entries)-1].Text) > 0) {
		sub.add(cached.container.EndTimestamp, "")
	}
	if cached.subtitlesByLang == nil {
		cached.subtitlesByLang = make(map[resource.Language]*Subtitles)
	}
	cached.subtitlesByLang[language] = sub
	return *sub
}

// NewCache returns a new instance.
func NewCache(cp text.Codepage, localizer resource.Localizer) *Cache {
	cache := &Cache{
		cp:        cp,
		localizer: localizer,

		movies: make(map[resource.Key]*cachedMovie),
	}
	return cache
}

// InvalidateResources lets the cache remove any movies from resources that are specified in the given slice.
func (cache *Cache) InvalidateResources(ids []resource.ID) {
	for _, id := range ids {
		for key := range cache.movies {
			if key.ID == id {
				delete(cache.movies, key)
			}
		}
	}
}

func (cache *Cache) cached(key resource.Key) (*cachedMovie, error) {
	value, existing := cache.movies[key]
	if existing {
		return value, nil
	}
	selector := cache.localizer.LocalizedResources(key.Lang)
	view, err := selector.Select(key.ID.Plus(key.Index))
	if err != nil {
		return nil, errors.New("no movie found")
	}
	if (view.ContentType() != resource.Movie) || view.Compound() || (view.BlockCount() != 1) {
		return nil, errors.New("invalid resource type")
	}
	reader, err := view.Block(0)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	container, err := Read(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	cached := &cachedMovie{
		cp:        cache.cp,
		container: container,
	}
	cache.movies[key] = cached
	return cached, nil
}

// Container retrieves and caches the underlying movie, and returns the complete container.
func (cache *Cache) Container(key resource.Key) (Container, error) {
	cached, err := cache.cached(key)
	if err != nil {
		return Container{}, err
	}
	return cached.container, nil
}

// Audio retrieves and caches the underlying movie, and returns the audio only.
func (cache *Cache) Audio(key resource.Key) (sound audio.L8, err error) {
	cached, err := cache.cached(key)
	if err != nil {
		return
	}
	return cached.audio(), nil
}

type Scene struct {
	Frames []bitmap.Bitmap
}

func (cache *Cache) Video(key resource.Key) ([]Scene, error) {
	cached, err := cache.cached(key)
	if err != nil {
		return nil, err
	}
	return cached.video(), nil
}

func (cache *Cache) Subtitles(key resource.Key, language resource.Language) (Subtitles, error) {
	cached, err := cache.cached(key)
	if err != nil {
		return Subtitles{}, err
	}
	return cached.subtitles(language), nil
}
