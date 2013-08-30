package main

import (
	"bufio"
	"fmt"
	"log"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"sync"
)

func extForLanguage(lang string) string {
	switch lang {
	case "c":
		return "c"
	case "golang":
		return "go"
	case "python":
		return "py"
	case "ruby":
		return "rb"
	}
	return ""
}

func (s *Server) streamOutput(stream io.ReadCloser, wg *sync.WaitGroup) {
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		log.Println("Dispatching to test:")
		text := scanner.Text()
		s.broker.Dispatch("test", Event{text})
	}
	if err := scanner.Err(); err != nil {
		// TODO: oops!
		//http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	wg.Done()
}

func (s *Server) runCodeHandler(w http.ResponseWriter, r *http.Request) {
	language := r.FormValue("language")
	body := r.FormValue("body")

	log.Printf("Running %s program...\n", language)

	ext := extForLanguage(language)
	fileName := fmt.Sprintf("prog.%s", ext)
	dir, err := ioutil.TempDir("", "code-")
	defer os.RemoveAll(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	f, err := os.Create(path.Join(dir, fileName))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := f.WriteString(body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	f.Close()

	dockerArgs := []string{
		"docker",
		"run",
		"-n=false",
		fmt.Sprintf("-v=%s:/code:ro", dir),
		"runner",
		language,
		path.Join("/code", fileName),
	}

	cmd := exec.Command("sudo", dockerArgs...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//stderr, err := cmd.StderrPipe()
	//if err != nil {
	//    http.Error(w, err.Error(), http.StatusInternalServerError)
	//    return
	//}

	if err := cmd.Start(); err != nil {
		log.Printf("Error starting docker: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Cmd.Wait() closes the fds, so we need to wait for reading to finish first
	var wg sync.WaitGroup
	wg.Add(1)
	go s.streamOutput(stdout, &wg)
	//go streamOutput(stderr, w, &wg)
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.broker.Dispatch("test", Event{"--> Execution complete"})
}

type Server struct {
	broker *Broker
}

func (s *Server) eventStreamHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("New SSE subscriber")

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

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // For Nginx

	ch := s.broker.Subscribe("test")
	defer func() {
		log.Println("Cleaning up connection")
		s.broker.Unsubscribe(ch, "test")
	}()

	closer := c.CloseNotify()

	for {
		select {
		case e := <-ch:
			log.Println("New SSE message", e.Body)
			if _, err := fmt.Fprintf(w, "data: %s\n\n", e.Body); err != nil {
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

func main() {
	s := &Server{broker: NewBroker()}
	http.HandleFunc("/run-code/", s.runCodeHandler)
	http.HandleFunc("/event-stream/", s.eventStreamHandler)
	http.ListenAndServe(":8080", nil)
}

