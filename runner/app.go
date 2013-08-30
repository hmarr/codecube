package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"sync"
	"io"
	"bufio"
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

func streamOutput(stream io.ReadCloser, w http.ResponseWriter, wg *sync.WaitGroup) {
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		w.Write([]byte(scanner.Text() + "\n"))
	}
	if err := scanner.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	wg.Done()
}

func handler(w http.ResponseWriter, r *http.Request) {
	language := r.FormValue("language")
	body := r.FormValue("body")

	fmt.Printf("Running %s program...\n", language)

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
		fmt.Printf("Error starting docker: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Cmd.Wait() closes the fds, so we need to wait for reading to finish first
	var wg sync.WaitGroup
	wg.Add(1)
	go streamOutput(stdout, w, &wg)
	//go streamOutput(stderr, w, &wg)
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}

