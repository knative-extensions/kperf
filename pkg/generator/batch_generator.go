package generator

import (
	"os"
	"time"
)

type Generator func(string, int) (string, string)
type PostGenerator func(string, string) error

type BatchGenerator struct {
	interval          time.Duration
	count             int
	counter           int
	batch             int
	concurrency       int
	namespaceList     []string
	generateFunc      Generator
	postGeneratorFunc PostGenerator

	indexChan     chan int
	finishedChan  chan int
	finishedCount int
	doneChan      chan bool
}

func NewBatchGenerator(interval time.Duration, count, batch int, concurrency int, namespaceList []string, generator Generator, postGenerator PostGenerator) *BatchGenerator {
	return &BatchGenerator{
		interval:          interval,
		count:             count,
		counter:           0,
		batch:             batch,
		concurrency:       concurrency,
		namespaceList:     namespaceList,
		generateFunc:      generator,
		postGeneratorFunc: postGenerator,

		indexChan:     make(chan int, batch*5),
		finishedChan:  make(chan int, batch*5),
		finishedCount: 0,
		doneChan:      make(chan bool),
	}
}

func (bg *BatchGenerator) Generate() {
	ticker := time.NewTicker(bg.interval)
	defer ticker.Stop()
	go bg.checkFinished()
	for i := 0; i < bg.concurrency; i++ {
		go bg.doGenerate()
	}
	for {
		select {
		case <-bg.doneChan:
			return
		case <-ticker.C:
			i := 0
			for bg.counter < bg.count && i < bg.batch {
				bg.indexChan <- bg.counter
				bg.counter++
				i++
			}
		}
	}

}
func (bg *BatchGenerator) doGenerate() {
	for {
		select {
		case <-bg.doneChan:
			return
		case index := <-bg.indexChan:
			ns := bg.namespaceList[index%len(bg.namespaceList)]
			ns, name := bg.generateFunc(ns, index)
			if bg.postGeneratorFunc(ns, name) != nil {
				os.Exit(1)
			}
			bg.finishedChan <- 1
		}
	}
}

func (bg *BatchGenerator) checkFinished() {
	for {
		select {
		case <-bg.finishedChan:
			bg.finishedCount++
			if bg.finishedCount >= bg.count {
				close(bg.doneChan)
			}
		}
	}
}
