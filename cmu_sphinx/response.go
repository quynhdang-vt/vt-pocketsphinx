package cmu_sphinx

import (
	"strconv"
	"github.com/veritone/go-lattice/lattice"
)

type Word struct {
	Word       string
	StartFrame int32
	EndFrame   int32
	AScr       int32
	LScr       int32
	LBack      int32
	PProb      int32
}

type Response struct {
	Words []Word
}

func (w Word) ToUtterance(index int) (lattice.Utterance, error) {
	startTimeMs := int(w.StartFrame * 10)
	endTimeMs := int(w.EndFrame * 10)	
	durationMs := endTimeMs - startTimeMs

	// Speechmatics confidence is a floating point value with 3 points of precision, ranging from (0,1]
	// e.g. 0.984
	// We convert this into an integer value ranging from 0-1000
	// e.g. 984
	confidenceRaw := 1
	confidenceInt := int(confidenceRaw * 1000)

	newUtterance := lattice.Utterance{
		Index:       index,
		StartTimeMs: startTimeMs,
		StopTimeMs:  endTimeMs,
		DurationMs:  durationMs,
		Words: lattice.UtteranceWords{
			&lattice.UtteranceWord{
				Word:       w.Word,
				Confidence: confidenceInt,

				// Since the full lattice isn't provided we use the default values for the following fields.
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
    lastFrame:=s.Words[ len(s.Words)-1 ].EndFrame
	for _, word := range responseToAppend.Words {
		word.StartFrame = word.StartFrame+lastFrame
		word.EndFrame = word.EndFrame+lastFrame
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
