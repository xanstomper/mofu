package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/xanstomper/mofu"
)

type Track struct {
	Title    string
	Artist   string
	Album    string
	Duration int
	Bitrate  int
}

type Playlist struct {
	Name   string
	Tracks []Track
}

type MusicPlayer struct {
	mofu.Minimal
	playlists    []Playlist
	currentPL    int
	selected     int
	playing      bool
	currentTime  int
	volume       int
	shuffle      bool
	repeatMode   int
	width        int
	height       int
	sortBy       int
}

func NewMusicPlayer() *MusicPlayer {
	p := &MusicPlayer{
		volume: 75,
		playlists: []Playlist{
			{
				Name: "Recently Played",
				Tracks: []Track{
					{Title: "Bohemian Rhapsody", Artist: "Queen", Album: "A Night at the Opera", Duration: 354, Bitrate: 320},
					{Title: "Stairway to Heaven", Artist: "Led Zeppelin", Album: "Led Zeppelin IV", Duration: 482, Bitrate: 320},
					{Title: "Hotel California", Artist: "Eagles", Album: "Hotel California", Duration: 391, Bitrate: 256},
					{Title: "Comfortably Numb", Artist: "Pink Floyd", Album: "The Wall", Duration: 382, Bitrate: 320},
					{Title: "Sweet Child O' Mine", Artist: "Guns N' Roses", Album: "Appetite for Destruction", Duration: 356, Bitrate: 256},
					{Title: "Smells Like Teen Spirit", Artist: "Nirvana", Album: "Nevermind", Duration: 301, Bitrate: 320},
					{Title: "Yesterday", Artist: "The Beatles", Album: "Help!", Duration: 125, Bitrate: 256},
					{Title: "Imagine", Artist: "John Lennon", Album: "Imagine", Duration: 187, Bitrate: 320},
				},
			},
			{
				Name: "Favorites",
				Tracks: []Track{
					{Title: "Wish You Were Here", Artist: "Pink Floyd", Album: "Wish You Were Here", Duration: 334, Bitrate: 320},
					{Title: "November Rain", Artist: "Guns N' Roses", Album: "Use Your Illusion I", Duration: 537, Bitrate: 320},
					{Title: "Under Pressure", Artist: "Queen & David Bowie", Album: "Hot Space", Duration: 248, Bitrate: 256},
				},
			},
			{
				Name: "Chill Vibes",
				Tracks: []Track{
					{Title: "Clair de Lune", Artist: "Debussy", Album: "Suite Bergamasque", Duration: 302, Bitrate: 320},
					{Title: "Gymnopédie No.1", Artist: "Erik Satie", Album: "Gymnopédies", Duration: 194, Bitrate: 256},
					{Title: "Nocturne Op.9 No.2", Artist: "Chopin", Album: "Nocturnes", Duration: 271, Bitrate: 320},
					{Title: "Moonlight Sonata", Artist: "Beethoven", Album: "Piano Sonata No.14", Duration: 360, Bitrate: 320},
				},
			},
		},
	}
	return p
}

func (p *MusicPlayer) currentPlaylist() *Playlist {
	return &p.playlists[p.currentPL]
}

func (p *MusicPlayer) currentTrack() *Track {
	pl := p.currentPlaylist()
	if p.selected >= 0 && p.selected < len(pl.Tracks) {
		return &pl.Tracks[p.selected]
	}
	return nil
}

func (p *MusicPlayer) formatTime(secs int) string {
	return fmt.Sprintf("%d:%02d", secs/60, secs%60)
}

func (p *MusicPlayer) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	p.width = r.Width
	p.height = r.Height

	leftW := 22
	rightW := r.Width - leftW

	// Playlists sidebar
	ctx.Renderer.WriteString(" Playlists", r.X, r.Y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)

	for i, pl := range p.playlists {
		y := r.Y + 1 + i
		style := mofu.DefaultStyle()
		prefix := "  "
		if i == p.currentPL {
			prefix = "▸ "
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
		}
		ctx.Renderer.WriteString(fmt.Sprintf("%s%s (%d)", prefix, pl.Name, len(pl.Tracks)), r.X, y, style.Foreground, style.Background, style.Attrs)
	}

	ctx.Renderer.WriteString("│", r.X+leftW-1, r.Y, mofu.Hex("444444"), mofu.ColorBlack, 0)

	// Track list
	pl := p.currentPlaylist()
	ctx.Renderer.WriteString(fmt.Sprintf(" %s", pl.Name), r.X+leftW, r.Y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)

	header := fmt.Sprintf(" #  %-30s %-20s %6s", "Title", "Artist", "Time")
	ctx.Renderer.WriteString(header, r.X+leftW, r.Y+1, mofu.Hex("6c7086"), mofu.ColorBlack, 0)
	ctx.Renderer.WriteString(strings.Repeat("─", rightW-1), r.X+leftW, r.Y+2, mofu.Hex("444444"), mofu.ColorBlack, 0)

	for i, track := range pl.Tracks {
		y := r.Y + 3 + i
		if y >= r.Y+r.Height-5 {
			break
		}

		title := track.Title
		if len(title) > 30 {
			title = title[:27] + "..."
		}
		artist := track.Artist
		if len(artist) > 20 {
			artist = artist[:17] + "..."
		}

		line := fmt.Sprintf(" %-3d%-30s %-20s %6s", i+1, title, artist, p.formatTime(track.Duration))
		if len(line) > rightW-1 {
			line = line[:rightW-1]
		}

		style := mofu.DefaultStyle()
		if i == p.selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", rightW-1), r.X+leftW, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		} else if p.playing && i == p.selected {
			style = mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1"))
		}

		ctx.Renderer.WriteString(line, r.X+leftW, y, style.Foreground, style.Background, style.Attrs)
	}

	// Now playing bar
	npY := r.Y + r.Height - 4
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, npY, mofu.Hex("444444"), mofu.ColorBlack, 0)

	track := p.currentTrack()
	if track != nil {
		icon := "■"
		if p.playing {
			icon = "▶"
		}

		npLine := fmt.Sprintf(" %s %s — %s [%s/%s]", icon, track.Title, track.Artist, p.formatTime(p.currentTime), p.formatTime(track.Duration))
		if len(npLine) > r.Width-2 {
			npLine = npLine[:r.Width-5] + "..."
		}
		ctx.Renderer.WriteString(npLine, r.X, npY+1, mofu.Hex("ff69b4"), mofu.ColorBlack, 0)

		// Progress bar
		if track.Duration > 0 {
			progW := r.Width - 4
			progress := float64(p.currentTime) / float64(track.Duration)
			filled := int(progress * float64(progW))
			bar := strings.Repeat("━", filled) + strings.Repeat("─", progW-filled)
			pct := fmt.Sprintf("%.0f%%", progress*100)
			ctx.Renderer.WriteString(" "+bar+" "+pct, r.X, npY+2, mofu.Hex("89b4fa"), mofu.ColorBlack, 0)
		}
	}

	// Volume + controls
	volBar := strings.Repeat("█", p.volume/10) + strings.Repeat("░", 10-p.volume/10)
	volLine := fmt.Sprintf(" Vol:[%s] %d%%", volBar, p.volume)

	controls := volLine + "  S:shuffle R:repeat"
	if len(controls) > r.Width-2 {
		controls = controls[:r.Width-2]
	}
	ctx.Renderer.WriteString(controls, r.X, r.Y+r.Height-1, mofu.Hex("585b70"), mofu.Hex("1e1e2e"), 0)
}

func (p *MusicPlayer) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke := event.Data.(mofu.KeyEvent)
	pl := p.currentPlaylist()

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()

	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if p.selected < len(pl.Tracks)-1 {
			p.selected++
		}

	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if p.selected > 0 {
			p.selected--
		}

	case ke.Key == mofu.KeyLeft:
		if p.currentPL > 0 {
			p.currentPL--
			p.selected = 0
		}

	case ke.Key == mofu.KeyRight:
		if p.currentPL < len(p.playlists)-1 {
			p.currentPL++
			p.selected = 0
		}

	case ke.Key == mofu.KeySpace:
		p.playing = !p.playing

	case ke.Key == mofu.KeyEnter:
		p.playing = true
		p.currentTime = 0

	case (len(ke.Runes) > 0 && ke.Runes[0] == 'n') && p.playing:
		if p.selected < len(pl.Tracks)-1 {
			p.selected++
			p.currentTime = 0
		}

	case (len(ke.Runes) > 0 && ke.Runes[0] == 'p') && p.playing:
		if p.selected > 0 {
			p.selected--
			p.currentTime = 0
		}

	case len(ke.Runes) > 0 && ke.Runes[0] == 's':
		p.shuffle = !p.shuffle

	case len(ke.Runes) > 0 && ke.Runes[0] == 'r':
		p.repeatMode = (p.repeatMode + 1) % 3

	case ke.Key == mofu.KeyUp && ke.Ctrl:
		if p.volume < 100 {
			p.volume += 5
		}

	case ke.Key == mofu.KeyDown && ke.Ctrl:
		if p.volume > 0 {
			p.volume -= 5
		}
	}

	if p.playing && p.selected < len(pl.Tracks) {
		p.currentTime++
		if p.currentTime >= pl.Tracks[p.selected].Duration {
			if p.repeatMode == 2 {
				p.currentTime = 0
			} else if p.selected < len(pl.Tracks)-1 {
				p.selected++
				p.currentTime = 0
			} else if p.repeatMode == 1 {
				p.selected = 0
				p.currentTime = 0
			} else {
				p.playing = false
			}
		}
	}

	return nil
}

func main() {
	rand.Seed(42)
	app := NewMusicPlayer()
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
