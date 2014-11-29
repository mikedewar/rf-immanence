package main

import (
	"bufio"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

// scan a range of the spectrum, emitting the result periodically. This just wraps http://kmkeen.com/rtl-power/index.html
func scan(outChan chan []byte, quitChan chan bool) {
	cmd := exec.Command("/usr/local/bin/rtl_power", "-f", "118M:150M:8k", "-g", "50", "-i", "5")
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	reader := bufio.NewReader(stdout)
	var t, τ time.Time
	data := make([]float64, 0)
	//2014-11-28, 13:23:14
	//Mon Jan 2 15:04:05 -0700 MST 2006
	layout := "2006-01-02 15:04:05"
	for {
	Concatenate:
		for {
			select {
			default:
				line, err := reader.ReadString('\n')
				if err != nil {
					log.Fatal(err)
				}
				lineArray := strings.Split(line, ",")
				t, err = time.Parse(layout, lineArray[0]+lineArray[1])
				if err != nil {
					log.Fatal(err)
				}
				if τ.IsZero() {
					τ = t
				}
				if t != τ {
					// these next few lines perform a local normalisation.
					// TODO make this normalise across the possible scale of values, rathe than just the current frame's observations.
					min := data[0]
					for _, value := range data {
						if value < min {
							min = value
						}
					}
					for i, _ := range data {
						data[i] -= min
					}
					max := data[0]
					for _, value := range data {
						if value > max {
							max = value
						}
					}
					scale := 255.0 / max
					for i, _ := range data {
						data[i] *= scale
					}
					// marshal and send for broadcasting
					out, err := json.Marshal(data)
					if err != nil {
						log.Fatal(err)
					}
					outChan <- out
					data = make([]float64, 0)
					τ = t
					break Concatenate
				}
				dstring := lineArray[7:]
				d := make([]float64, len(dstring))
				for i, ds := range dstring {
					d[i], err = strconv.ParseFloat(strings.TrimSpace(ds), 64)
					if err != nil {
						log.Fatal(err)
					}
				}
				data = append(data, d...)
				τ = t
			case <-quitChan:
				return
			}
		}
	}
}

func main() {

	wsServer := &Server{
		addConn:   make(chan *connection),
		delConn:   make(chan *connection),
		broadcast: make(chan []byte),
		emitChan:  make(chan []byte),
	}

	c := make(chan os.Signal, 1)
	quitChan := make(chan bool)
	signal.Notify(c, os.Interrupt, os.Kill)

	go wsServer.websocketRouter()
	go scan(wsServer.broadcast, quitChan)

	http.Handle("/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/ws", wsServer.websocketHandler)
	log.Println("serving rtl_power websocket on 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))

	<-c
	quitChan <- true

}
