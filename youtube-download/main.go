package main

import (
	"flag"
	"fmt"
	"log"
	"os/user"

	"github.com/kkdai/youtube"
)

func main() {
	var youtubeURL string
	var filename string
	flag.StringVar(&youtubeURL, "youtubeURL", "youtube video url", "Url of video")
	flag.StringVar(&filename, "filename", "dl.mp4", "filename for file")
	flag.Parse()
	log.Println(flag.Args())
	usr, _ := user.Current()
	currentDir := fmt.Sprintf("%v/Movies/youtubedr", usr.HomeDir)
	log.Println("download to dir=", currentDir)
	y := youtube.NewYoutube(true, true)
	y.DecodeURL(youtubeURL)
	//arg := flag.Arg(0)
	if err := y.DecodeURL(youtubeURL); err != nil {
		fmt.Println("err:", err)
	}
	if err := y.StartDownload(currentDir, "dl.mp4", "100", 0); err != nil {
		fmt.Println("err:", err)
	}
}
