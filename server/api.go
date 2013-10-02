package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	dcli "github.com/fsouza/go-dockerclient"
	"github.com/garyburd/redigo/redis"
	"io"
	"log"
	"net/http"
	"time"
)

func main() {
	redisPool := redis.NewPool(func() (redis.Conn, error) {
		return redis.Dial("tcp", ":6379")
	}, 5)

	s := &Server{
		broker:    NewBroker(),
		uidPool:   NewUidPool(20000, 25000),
		redisPool: redisPool,
	}
	http.HandleFunc("/run-snippet/", s.runSnippetHandler)
	http.HandleFunc("/load-snippet/", s.loadSnippetHandler)
	http.HandleFunc("/event-stream/", s.eventStreamHandler)
	http.ListenAndServe(":8080", nil)
}

type Server struct {
	broker    *Broker
	uidPool   *UidPool
	redisPool *redis.Pool
}

type Snippet struct {
	Language string `json:"language"`
	Code     string `json:"code"`
}

func (s *Server) saveSnippet(id string, sn *Snippet) error {
	conn := s.redisPool.Get()
	defer conn.Close()

	_, err := conn.Do("SET", fmt.Sprintf("cc:language:%s", id), sn.Language)
	if err != nil {
		return err
	}

	_, err = conn.Do("SET", fmt.Sprintf("cc:code:%s", id), sn.Code)
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) loadSnippet(id string) (*Snippet, error) {
	conn := s.redisPool.Get()
	defer conn.Close()

	key := fmt.Sprintf("cc:language:%s", id)
	language, err := redis.String(conn.Do("GET", key))
	if err != nil {
		return nil, err
	}

	key = fmt.Sprintf("cc:code:%s", id)
	code, err := redis.String(conn.Do("GET", key))
	if err != nil {
		return nil, err
	}

	return &Snippet{Language: language, Code: code}, nil
}

func (s *Server) streamOutput(id string, streamName string, stream io.Reader) {
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		text := scanner.Text()
		s.broker.Dispatch(id, Event{text})
	}
	if err := scanner.Err(); err != nil {
		// TODO: something?
	}
}

func (s *Server) runSnippetHandler(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	language := r.FormValue("language")
	code := r.FormValue("code")

	log.Printf("Running code for id %s\n", id)
	err := s.saveSnippet(id, &Snippet{Language: language, Code: code})
	if err != nil {
		log.Printf("[E] Error saving snippet %s: %s\n", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	runner := NewRunner(dockerClient(), language, code)
	runner.UidPool = s.uidPool
	outReader, outWriter := io.Pipe()
	errReader, errWriter := io.Pipe()
	runner.OutStream = outWriter
	runner.ErrStream = errWriter

	go s.streamOutput(id, "stdout", outReader)
	go s.streamOutput(id, "stderr", errReader)

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
	s.broker.Dispatch(id, Event{msg})

	outReader.Close()
	errReader.Close()
}

func (s *Server) loadSnippetHandler(w http.ResponseWriter, r *http.Request) {
	var snippet *Snippet

	id := r.FormValue("id")
	if id != "" {
		snippet, _ = s.loadSnippet(string(id))
	}

	json, err := json.Marshal(snippet)
	if err != nil {
		log.Printf("[E] Error marshalling json: %s\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(json))
}

func (s *Server) eventStreamHandler(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	log.Printf("New SSE subscriber (%s)\n", id)

	if id == "" {
		http.Error(w, "id can't be blank", http.StatusBadRequest)
		return
	}

	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusBadRequest)
		return
	}

	c, ok := w.(http.CloseNotifier)
	if !ok {
		http.Error(w, "close notification unsupported", http.StatusBadRequest)
		return
	}
	closer := c.CloseNotify()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // For Nginx

	ch := s.broker.Subscribe(id)
	defer func() {
		log.Println("Cleaning up connection")
		s.broker.Unsubscribe(ch, id)
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
