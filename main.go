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
)

// scan a range of the spectrum, emitting the result periodically. This just wraps http://kmkeen.com/rtl-power/index.html
func scan(outChan chan []byte, quitChan chan bool) {
	cmd := exec.Command("/usr/local/bin/rtl_power", "-f", "118M:137M:8k", "-g", "50", "-i", "10")
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	reader := bufio.NewReader(stdout)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		dstring := strings.Split(line, ",")[7:]
		d := make([]float64, len(dstring))
		for i, ds := range dstring {
			d[i], err = strconv.ParseFloat(strings.TrimSpace(ds), 64)
			if err != nil {
				log.Fatal(err)
			}
		}
		out, err := json.Marshal(d)
		if err != nil {
			log.Fatal(err)
		}
		select {
		case outChan <- out:
		case <-quitChan:
			return
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

	http.HandleFunc("/ws", wsServer.websocketHandler)
	log.Println("serving rtl_power websocket on 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))

	<-c
	quitChan <- true

}
