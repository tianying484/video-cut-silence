package main

import "os"

var (
	videoFolder = "/opt/source"
	splitFolder = "/opt/cut-video"
	mergeFile   = splitFolder + "/merge.txt"
	ffmpegPath  = "/usr/local/bin/ffmpeg"
)

func main() {
	err := os.MkdirAll(splitFolder, 0775)
	CheckError(err)

	v := Video{
		videos:    make([]string, 1000),
		processed: make(map[string]bool, 1000),
	}
	v.RemoveSilenceMoment()
}
