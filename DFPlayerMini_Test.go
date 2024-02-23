package DFPlayerMini

import (
	"fmt"
	"testing"
	"time"

	"go.bug.st/serial"
)

const (
	minTrackPlaybackTime   = time.Duration(time.Second * 3)
	trackPlaytimeIncrement = time.Duration(time.Second)
)

func TestAllFolderSequentialPlayback(t *testing.T) {
	sp, err := serial.Open("/dev/ttyUSB1", &serial.Mode{BaudRate: 9600, DataBits: 8, Parity: serial.NoParity, StopBits: serial.OneStopBit})
	if err != nil {
		println(fmt.Sprintf("serial.Open() failed: %v", err))
		panic(err)
	}

	sp.SetReadTimeout(time.Millisecond * 10)
	defer sp.Close()

	player := New(sp, DebugQuiet)

	time.Sleep(time.Millisecond * 500)

	for {
	INIT:
		println("SD Playback source")
		player.SelectPlaybackSource(PlaybackSourceSD)

		println("EQ set")
		player.SetEQ(EqRock)

		println("Stop repeat playback")
		player.StopRepeatPlayback()

		time.Sleep(time.Millisecond * 500)
		player.Discard()

		tracks, ok := player.GetSDTrackCount()
		if ok {
			println(fmt.Sprintf("Total SD tracks: %d", tracks))
		} else {
			println("GetSDTrackCount() failed")
			time.Sleep(time.Second)
			goto INIT
		}

		player.SetVolume(0)
		player.StopDAC()

		time.Sleep(time.Millisecond * 500)

		println("Building folder playlist...")
		folders, totalTracks := player.BuildFolderPlaylist()
		println(fmt.Sprintf("Numerical folders: %02d containing %02d tracks", len(folders), totalTracks))

		player.StartDAC()
		player.SetAmplificationGain(true, 2)
		player.SetVolume(1)

		progress := 0
		statusErrorCount := 0
		for folder, folderTracks := range folders {
			trackRuntime := time.Duration(time.Second * 0)
			currentTrack := 0
			for {
				currentTrack++
				if currentTrack > int(folderTracks) {
					break
				}
				progress++
				player.PlayFolderTrack(folder, uint8(currentTrack))
				for {
					cmd, param, ok := player.QueryStatus()
					if ok {
						switch cmd {
						case ErrorCondition:
							if param == ErrorTrackOutOfScope || param == ErrorTrackNotFound {
								println(fmt.Sprintf("Track not found (folder: %02d, track: %02d)", folder, currentTrack))
								goto NEXT
							}
						case MediaOut:
							println("Media removed. Re-initializing...")
							goto INIT

						case SdTrackFinished:
							if trackRuntime > minTrackPlaybackTime {
								println(fmt.Sprintf("SD card track #%04d finished playing", param))
								trackRuntime = time.Duration(time.Second * 0)
								goto NEXT
							}

						case GetStatus:
							if (param & 0x00FF) == TrackPlaying {
								if trackRuntime == 0 {
									println(fmt.Sprintf("Playing SD folder %02d, track: %0d/%0d", folder, progress, totalTracks))
								}
								trackRuntime = trackRuntime + trackPlaytimeIncrement
							}
						}
					} else {
						statusErrorCount++
						println(fmt.Sprintf("QueryStatus() error count: %02d", statusErrorCount))
					}
					time.Sleep(trackPlaytimeIncrement)
				}
			NEXT:
				time.Sleep(time.Millisecond * 0)
			}
		}
		break
	}

	println("Closing MP3 module")

	player.SetVolume(0)
	player.Stop()
	player.StopDAC()
	player.Sleep()
}
