package processing

import log "github.com/sirupsen/logrus"

type FroxlorProcessor struct {
}

func (fp *FroxlorProcessor) Process(domains []string) {
	log.Infof("Froxlor received domains %v", domains)
}

func (fp *FroxlorProcessor) ID() string {
	return "froxlor"
}