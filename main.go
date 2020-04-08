package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/radovskyb/watcher"
	"gopkg.in/yaml.v3"
)

var (
	homeDir  string
)

var (
	config = flag.String("c", ".wdf.stub.yaml", "")
	watch  = flag.Bool("w", false, "")
)

func init() {
	dir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	homeDir = dir

	flag.Parse()
}

func main() {
	file, err := ioutil.ReadFile(*config)
	if err != nil {
		panic(err)
	}

	var configNode yaml.Node
	if err := yaml.Unmarshal(file, &configNode); err != nil {
		panic(err)
	}

	var prevKey string
	fileMaps := make(map[string]string)
	for _, node := range configNode.Content {
		for i, n := range node.Content {
			if i%2 == 0 {
				prevKey = n.Value
				continue
			}
			var name string = tildeToHomeDir(n.Value)
			fileMaps[name] = prevKey
		}
	}

	for s, t := range fileMaps {
		if _, err := copy(s, t); err != nil {
			panic(err)
		}
		log.Printf("%s file was updated.\n", t)
	}

	if *watch {
		w := watcher.New()
		go w.Wait()
		for source, _ := range fileMaps {
			w.Add(source)
		}
		go func () {
			for {
				select {
				case event := <-w.Event:
					if event.Op == watcher.Write {
						to := fileMaps[event.Path]
						if _, err := copy(event.Path, to); err != nil {
							log.Fatalln(err)
						}
						log.Printf("%s was updated.\n", to)
					}
				case err := <-w.Error:
					log.Fatalln(err)
				case <-w.Closed:
					return
				}
			}
		}()
		if err := w.Start(time.Millisecond * 100); err != nil {
			log.Fatalln(err)
		}
	}
}

func copy(from, to string) (bool, error) {
	finfo, err := os.Stat(from)
	if err != nil {
		return false, err
	}
	source, err := ioutil.ReadFile(from)
	if err != nil {
		return false, err
	}
	if err := ioutil.WriteFile(to, source, finfo.Mode()); err != nil {
		return false, err
	}
	return true, nil
}

// Replaces tilde prefixes to home directory.
func tildeToHomeDir(name string) string {
	if strings.HasPrefix(name, "~/") {
		return path.Join(homeDir, name[2:])
	}
	return name
}
