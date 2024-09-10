package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	webclipboard "web-clipboard"
)

var listen = flag.String("listen", "0.0.0.0:80", "http server listen addr")
var fileDir = flag.String("file_dir", "file", "file dir")
var showLength = flag.Int("show_length", 10000, "show_length")

func main() {
	flag.Parse()
	parseEnv()

	s := webclipboard.NewServer(*listen, *showLength, *fileDir)
	s.Run()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, os.Kill, syscall.SIGTERM)
	signal := <-quit
	fmt.Printf("receive signal %v,quit... \n", signal)
}

func parseEnv() {
	if lis := os.Getenv("listen"); len(lis) > 0 {
		listen = &lis
	}
	if dir := os.Getenv("file_dir"); len(dir) > 0 {
		fileDir = &dir
	}
}
