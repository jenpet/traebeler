package processing

import (
	"github.com/jenpet/traebeler/processing/froxlor"
)

type processor interface {
	Process(domains []string)
	ID() string
	Init() error
}

var repo processorRepository

type processorRepository map[string]processor

func (pr *processorRepository) GetProcessor(id string) processor {
	return (*pr)[id]
}

func Repository() *processorRepository {
	if repo == nil {
		repo = map[string]processor{}
	}
	repo["froxlor"] = &froxlor.Processor{}
	return &repo
}