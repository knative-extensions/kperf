package generator

type Clean func(ns, name string)

type BatchCleaner struct {
	namespaceNameList [][2]string
	concurrency       int
	cleanFunc         Clean

	doneChan          chan bool
	namespaceNameChan chan [2]string
	finishedChan      chan int
	finishedCount     int
}

func NewBatchCleaner(namespaceNameList [][2]string, concurrency int, cleanFunc Clean) *BatchCleaner {
	return &BatchCleaner{
		namespaceNameList: namespaceNameList,
		concurrency:       concurrency,
		namespaceNameChan: make(chan [2]string, len(namespaceNameList)),
		cleanFunc:         cleanFunc,

		doneChan:      make(chan bool),
		finishedChan:  make(chan int, concurrency*5),
		finishedCount: 0,
	}
}

func (bc *BatchCleaner) Clean() {

	go bc.checkFinished()
	for i := 0; i < bc.concurrency; i++ {
		go bc.doClean()
	}
	for _, nsname := range bc.namespaceNameList {
		bc.namespaceNameChan <- nsname
	}
	<-bc.doneChan

}

func (bc *BatchCleaner) doClean() {
	for {
		select {
		case <-bc.doneChan:
			return
		case nsname := <-bc.namespaceNameChan:
			bc.cleanFunc(nsname[0], nsname[1])
			bc.finishedChan <- 1
		}
	}
}

func (bc *BatchCleaner) checkFinished() {
	for {
		select {
		case <-bc.finishedChan:
			bc.finishedCount++
			if bc.finishedCount >= len(bc.namespaceNameList) {
				close(bc.doneChan)
			}
		}
	}
}
