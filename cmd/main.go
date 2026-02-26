package main

import (
	"sync"
	"time"

	"flomation.app/automate/runner/internal/config"
	"flomation.app/automate/runner/internal/runner"
	"flomation.app/automate/runner/internal/version"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.WithFields(log.Fields{
		"version": version.Version,
		"hash":    version.GetHash(),
		"date":    version.BuiltDate,
	}).Info("Starting Flomation Runner")

	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("unable to load config")
		return
	}

	r, err := runner.NewService(cfg)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("unable to start runner")
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func(w *sync.WaitGroup, run *runner.Service) {
		for {
			if !r.IsRunning() {
				wg.Done()
				break
			}

			time.Sleep(time.Second * 5)
		}
	}(&wg, r)

	wg.Wait()

}
