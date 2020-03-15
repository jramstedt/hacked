package edit

import (
	"github.com/inkyblackness/hacked/ss1/content/audio"
	"github.com/inkyblackness/hacked/ss1/content/movie"
	"github.com/inkyblackness/hacked/ss1/content/text"
	"github.com/inkyblackness/hacked/ss1/edit/media"
	"github.com/inkyblackness/hacked/ss1/resource"
)

const audioEntrySize = 0x2000

// MovieService provides read/write functionality.
type MovieService struct {
	cp text.Codepage

	movieViewer media.MovieViewerService
	movieSetter media.MovieSetterService
}

// NewMovieService returns a new instance based on given accessor.
func NewMovieService(codepage text.Codepage,
	movieViewer media.MovieViewerService, movieSetter media.MovieSetterService) MovieService {
	return MovieService{
		cp: codepage,

		movieViewer: movieViewer,
		movieSetter: movieSetter,
	}
}

// RestoreFunc creates a snapshot of the current movie and returns a function to restore it.
func (service MovieService) RestoreFunc(key resource.Key) func(setter media.MovieBlockSetter) {
	oldContainer, _ := service.movieViewer.Container(key)
	isModified := service.movieViewer.Modified(key)

	return func(setter media.MovieBlockSetter) {
		if isModified {
			service.movieSetter.Set(setter, key, oldContainer)
		} else {
			service.movieSetter.Remove(setter, key)
		}
	}
}

// Remove erases the movie from the resources.
func (service MovieService) Remove(setter media.MovieBlockSetter, key resource.Key) {
	service.movieSetter.Remove(setter, key)
}

// Video returns the video component of identified movie.
func (service MovieService) Video(key resource.Key) []movie.Scene {
	return service.movieViewer.Video(key)
}

// RemoveScene cuts out the given scene from the movie.
func (service MovieService) RemoveScene(setter media.MovieBlockSetter, key resource.Key, scene int) {
	// TODO
}

// Audio returns the audio component of identified movie.
func (service MovieService) Audio(key resource.Key) audio.L8 {
	return service.movieViewer.Audio(key)
}

// SetAudio sets the audio component of identified movie.
func (service MovieService) SetAudio(setter media.MovieBlockSetter, key resource.Key, soundData audio.L8) {
	baseContainer := service.getBaseContainer(key)
	var filteredEntries []movie.Entry

	for _, entry := range baseContainer.Entries {
		if entry.Type() == movie.DataTypeAudio {
			continue
		}
		filteredEntries = append(filteredEntries, entry)
	}

	var audioEntries []movie.Entry
	startOffset := 0
	for (startOffset + audioEntrySize) <= len(soundData.Samples) {
		ts := movie.TimestampFromSeconds(float32(startOffset) / soundData.SampleRate)
		endOffset := startOffset + audioEntrySize
		audioEntries = append(audioEntries, movie.AudioEntry{
			EntryBase: movie.EntryBase{Time: ts},
			Samples:   soundData.Samples[startOffset:endOffset],
		})
		startOffset = endOffset
	}
	if startOffset < len(soundData.Samples) {
		ts := movie.TimestampFromSeconds(float32(startOffset) / soundData.SampleRate)
		audioEntries = append(audioEntries, movie.AudioEntry{
			EntryBase: movie.EntryBase{Time: ts},
			Samples:   soundData.Samples[startOffset:],
		})
	}
	endTimestamp := movie.TimestampFromSeconds(float32(len(soundData.Samples)) / soundData.SampleRate)

	var newEntries []movie.Entry
	lastInsertIndex := -1
	for _, filteredEntry := range filteredEntries {
		for _, audioEntry := range audioEntries[lastInsertIndex+1:] {
			if audioEntry.Timestamp().IsAfter(filteredEntry.Timestamp()) {
				break
			}
			newEntries = append(newEntries, audioEntry)
			lastInsertIndex++
		}
		newEntries = append(newEntries, filteredEntry)
	}
	baseContainer.AudioSampleRate = uint16(soundData.SampleRate)
	if endTimestamp.IsAfter(baseContainer.EndTimestamp) {
		baseContainer.EndTimestamp = endTimestamp
	}

	baseContainer.Entries = newEntries
	service.movieSetter.Set(setter, key, baseContainer)
}

// Subtitles returns the subtitles associated with the given key.
func (service MovieService) Subtitles(key resource.Key, language resource.Language) movie.Subtitles {
	return service.movieViewer.Subtitles(key, language)
}

// SetSubtitles sets the subtitles of identified movie in given language.
func (service MovieService) SetSubtitles(setter media.MovieBlockSetter, key resource.Key,
	language resource.Language, subtitles movie.Subtitles) {
	baseContainer := service.getBaseContainer(key)
	var filteredEntries []movie.Entry
	subtitleControl := movie.SubtitleControlForLanguage(language)
	areaIndex := -1

	for _, entry := range baseContainer.Entries {
		subtitleEntry, isSubtitle := entry.(movie.SubtitleEntry)
		if !isSubtitle {
			filteredEntries = append(filteredEntries, entry)
			continue
		}
		if subtitleEntry.Control == subtitleControl {
			continue
		}
		if subtitleEntry.Control == movie.SubtitleArea {
			areaIndex = len(filteredEntries)
		}
		filteredEntries = append(filteredEntries, entry)
	}

	var newEntries []movie.Entry
	if areaIndex < 0 {
		// Ensure a subtitle area is defined.
		// The area is hardcoded. While the engine respects any area, placing the text in the
		// frame area will have the pixels become overwritten. As such, there are many "wrong" options,
		// and only a few right ones. There's no need to make them editable.
		newEntries = append(newEntries,
			movie.SubtitleEntry{
				EntryBase: movie.EntryBase{},
				Control:   movie.SubtitleArea,
				Text:      service.cp.Encode("20 365 620 395 CLR"),
			})
	}
	lastInsertIndex := -1
	for filteredIndex, filteredEntry := range filteredEntries {
		if filteredIndex > areaIndex && filteredEntry.Type() != movie.DataTypePaletteReset && filteredEntry.Type() != movie.DataTypePalette &&
			filteredEntry.Type() != movie.DataTypeAudio {
			for _, subEntry := range subtitles.Entries[lastInsertIndex+1:] {
				if subEntry.Timestamp.IsAfter(filteredEntry.Timestamp()) {
					break
				}

				newEntries = append(newEntries,
					movie.SubtitleEntry{
						EntryBase: movie.EntryBase{Time: subEntry.Timestamp},
						Control:   subtitleControl,
						Text:      service.cp.Encode(subEntry.Text),
					})
				lastInsertIndex++
			}
		}
		newEntries = append(newEntries, filteredEntry)
	}

	baseContainer.Entries = newEntries
	service.movieSetter.Set(setter, key, baseContainer)
}

func (service MovieService) getBaseContainer(key resource.Key) movie.Container {
	container, err := service.movieViewer.Container(key)
	if err != nil {
		container = movie.Container{
			VideoWidth:      600,
			VideoHeight:     300,
			AudioSampleRate: 22050,
		}
	}
	return container
}
