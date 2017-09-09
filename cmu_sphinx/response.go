package cmu_sphinx

import (
	"github.com/veritone/go-lattice/lattice"
	"strconv"
)

type Word struct {
	Word       string
	StartTime int32
	EndTime   int32
	AScr       int32
	LScr       int32
	LBack      int32
	PProb      int32
	Confidence       float64
}

type Response struct {
	Words []Word
	// Also Hypothesis
	Hypotheses [] string
}

func (w Word) ToUtterance(index int) (lattice.Utterance, error) {

	durationMs := w.EndTime - w.StartTime


	confidenceInt := int(w.Confidence * 1000)
	newUtterance := lattice.Utterance{
		Index:       index,
		StartTimeMs: int(w.StartTime),
		StopTimeMs:  int(w.EndTime),
		DurationMs:  int(durationMs),
		Words: lattice.UtteranceWords{
			&lattice.UtteranceWord{
				Word:       w.Word,
				Confidence: confidenceInt,
				BestPathForward:  true,
				BestPathBackward: true,
				SpanningForward:  false,
				SpanningBackward: false,
				SpanningLength:   1,
			},
		},
	}

	return newUtterance, nil
}

func (s *Response) Append(responseToAppend Response) error {
	if len(s.Words) == 0 {
		// The current response is empty, append directly
		s.Words = append(s.Words, responseToAppend.Words...)
		return nil
	}
	// get the last endFrame?
	lastTime := s.Words[len(s.Words)-1].EndTime
	for _, word := range responseToAppend.Words {
		word.StartTime = word.StartTime + lastTime
		word.EndTime = word.EndTime + lastTime
		s.Words = append(s.Words, word)
	}

	return nil
}

func (s Response) ToLattice() (lattice.Lattice, error) {
	newLattice := make(lattice.Lattice, len(s.Words))

	for i, word := range s.Words {
		newUtterance, err := word.ToUtterance(i)
		if err != nil {
			return nil, err
		}

		newLattice[strconv.Itoa(i)] = &newUtterance
	}

	return newLattice, nil
}
