package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Video struct {
	videos    []string
	processed map[string]bool
}

func (v *Video) RemoveSilenceMoment() {
	v.ScanVideos()
	for _, video := range v.videos {
		if v.processed[video] {
			err := os.Remove(video)
			CheckError(err)
			continue
		}

		log.Println(video)
		moments := v.DetectSilenceMoments(video)

		var (
			start = 0.0
			index = 0
		)
		for k, moment := range moments {
			if k != len(moments)-1 && moment.duration < 10.0 {
				continue
			}

			log.Println(moment)

			//split
			v.SplitVideos(video, index, start, moment.begin-1)
			index++

			v.SplitVideos(video, index, moment.begin, moment.end)
			index++

			start = moment.end

			// merge
			if k == len(moments)-1 {
				v.MergeVideos(video, index)
			}
		}

		var answer string
		fmt.Println("Please enter any key to continue. ")
		_, _ = fmt.Scanln(&answer)
		_ = answer

		//remove temp split videos
		for i := 0; i < index; i++ {
			err := os.Remove(fmt.Sprintf("%s/output_%02d.mp4", splitFolder, i))
			CheckError(err)
		}
	}
}

func (v *Video) MergeVideos(video string, index int) {
	newVideo := strings.Replace(video, ".mp4", "-ok.mp4", 1)
	sb := strings.Builder{}
	sb.Grow(1000)
	for i := 0; i < index; i++ {
		if i%2 != 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf("file 'output_%02d.mp4'\n", i))
	}
	err := ioutil.WriteFile(mergeFile, []byte(sb.String()), 0777)
	CheckError(err)

	//ffmpeg -i "concat:%s" -c copy %s
	cmdline := fmt.Sprintf("%s -y -f concat -i %s -c copy '%s'", ffmpegPath, mergeFile, newVideo)
	log.Println(cmdline)
	cmd := exec.Command("sh", "-c", cmdline)
	err = cmd.Start()
	CheckError(err)

	err = cmd.Wait()
	CheckError(err)
}

func (v *Video) SplitVideos(video string, index int, start, end float64) {
	cmdline := fmt.Sprintf("%s -y -i '%s' -ss %f -to %f -c copy %s/output_%02d.mp4", ffmpegPath, video, start, end, splitFolder, index)
	log.Println(cmdline)
	cmd := exec.Command("sh", "-c", cmdline)
	err := cmd.Start()
	CheckError(err)

	err = cmd.Wait()
	CheckError(err)
}

func (v *Video) DetectSilenceMoments(video string) []SilenceMoment {
	cmdline := fmt.Sprintf(" -i '"+video+"' -af \"silencedetect\" -vn -sn -dn -f null /dev/null", ffmpegPath)

	var b bytes.Buffer
	cmd := exec.Command("sh", "-c", cmdline)
	cmd.Stdout = &b
	cmd.Stderr = &b

	err := cmd.Start()
	CheckError(err)

	err = cmd.Wait()
	CheckError(err)

	lines := strings.Split(b.String(), "\n")

	sb := strings.Builder{}
	sb.Grow(1000)

	for _, line := range lines {
		if !strings.Contains(line, "silencedetect") {
			continue
		}
		sb.WriteRune('\n')
		sb.WriteString(line)
	}
	silenceParts := make([]string, 0, 100)

	fields := strings.Fields(sb.String())
	for k, v := range fields {
		if v == "silence_start:" || v == "silence_end:" || v == "silence_duration:" {
			//log.Println(k, v)
			t := strings.ReplaceAll(fields[k+1], "[silencedetect", "")
			//log.Println(k+1, fields[k+1], t)
			silenceParts = append(silenceParts, t)
		}
	}

	moments := make([]SilenceMoment, 0, len(silenceParts)/3)
	for i := 0; i < len(silenceParts); i += 3 {
		begin, _ := strconv.ParseFloat(silenceParts[i], 64)
		end, _ := strconv.ParseFloat(silenceParts[i+1], 64)
		duration, _ := strconv.ParseFloat(silenceParts[i+2], 64)
		moments = append(moments, SilenceMoment{
			begin:    begin,
			end:      end,
			duration: duration,
		})
	}

	return moments
}

func (v *Video) ScanVideos() {
	err := filepath.Walk(videoFolder, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".mp4") {
			return nil
		}
		if strings.Contains(info.Name(), "ok") {
			done := strings.Replace(path, "-ok.mp4", ".mp4", 1)
			v.processed[done] = true
			return nil
		}

		v.videos = append(v.videos, path)
		return nil
	})

	CheckError(err)
}
