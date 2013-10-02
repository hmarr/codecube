package main

import (
	"fmt"
	"log"
	"net/http"
	"io"
	"bufio"
	"time"
	dcli "github.com/fsouza/go-dockerclient"
)

func main() {
	s := &Server{broker: NewBroker(), uidPool: NewUidPool(20000, 25000)}
	http.HandleFunc("/run-code/", s.runCodeHandler)
	http.HandleFunc("/event-stream/", s.eventStreamHandler)
	http.ListenAndServe(":8080", nil)
}

type Server struct {
	broker  *Broker
	uidPool *UidPool
}

func (s *Server) streamOutput(streamName string, stream io.Reader) {
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		text := scanner.Text()
		s.broker.Dispatch("test", Event{text})
	}
	if err := scanner.Err(); err != nil {
		// TODO: something?
	}
}

func (s *Server) runCodeHandler(w http.ResponseWriter, r *http.Request) {
	language := r.FormValue("language")
	code := r.FormValue("body")

	log.Printf("Running %s program...\n", language)

	runner := NewRunner(dockerClient(), language, code)
	runner.UidPool = s.uidPool
	outReader, outWriter := io.Pipe()
	errReader, errWriter := io.Pipe()
	runner.OutStream = outWriter
	runner.ErrStream = errWriter

	go s.streamOutput("stdout", outReader)
	go s.streamOutput("stderr", errReader)

	log.Println("Running code...")
	status, err := runner.Run(10000)
	if err != nil {
		log.Printf("[E] Error running code: %s\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var msg string
	if status == STATUS_TIMED_OUT {
		msg = "=> timed out after 10s"
	} else {
		msg = fmt.Sprintf("=> exited with status %d", status)
	}
	s.broker.Dispatch("test", Event{msg})

	outReader.Close()
	errReader.Close()
}

func (s *Server) eventStreamHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("New SSE subscriber")
	log.Println(r.URL.Path[1])

	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	c, ok := w.(http.CloseNotifier)
	if !ok {
		http.Error(w, "close notification unsupported", http.StatusInternalServerError)
		return
	}
	closer := c.CloseNotify()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // For Nginx

	ch := s.broker.Subscribe("test")
	defer func() {
		log.Println("Cleaning up connection")
		s.broker.Unsubscribe(ch, "test")
	}()

	for {
		select {
		case e := <-ch:
			log.Println("New SSE message", e.Body)
			if _, err := fmt.Fprintf(w, "data: %s\n\n", e.Body); err != nil {
				log.Println("Connection not writeable")
				return
			}
			f.Flush()
		case <-time.After(1e9 * 15):
			if _, err := fmt.Fprintf(w, ":keepalive\n\n"); err != nil {
				log.Println("Connection not writeable")
				return
			}
			f.Flush()
		case <-closer:
			log.Println("Connection closed")
			return
		}
	}
	log.Println("SSE done")
}

func dockerClient() *dcli.Client {
	client, err := dcli.NewClient("http://127.0.0.1:4243")
	if err != nil {
		panic(err)
	}
	return client
}

