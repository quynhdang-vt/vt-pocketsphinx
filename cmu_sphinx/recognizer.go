package cmu_sphinx

import (
	"encoding/binary"
	"github.com/xlab/pocketsphinx-go/pocketsphinx"
	"github.com/xlab/pocketsphinx-go/sphinx"
	"log"
	"time"
)

type Recognizer struct {
	inSpeech          bool
	uttStarted        bool
	dec               *sphinx.Decoder
	infileName        *string
	textHypothesis    string
	speech, cpu, wall time.Duration
	Response          Response
}

func ConvertToInt16ArrayInLittleEndian(raw []byte, rawLen int) []int16 {
	const SIZEOF_INT16 = 2
	data := make([]int16, rawLen/SIZEOF_INT16)
	for i := range data {
		data[i] = int16(binary.LittleEndian.Uint16(raw[i*SIZEOF_INT16 : (i+1)*SIZEOF_INT16]))
	}
	//	log.Printf("convertToInt16Array...from %d bytes to  %d int\n", rawLen, len(data))
	return data
}

func (l *Recognizer) Process(raw []byte, nRaw int) (frames int32) {
	//	log.Println("process .. ", nRaw)
	// ProcessRaw with disabled search because callback needs to be relatime
	frames, ok := l.dec.ProcessRaw(ConvertToInt16ArrayInLittleEndian(raw, nRaw), false, false)
	// log.Printf("processed: %d frames, ok: %v", frames, ok)
	if !ok {
		log.Fatal("??? why is it not ok?")
	}
	return frames
}

func (l *Recognizer) Report() {
	hyp, _ := l.dec.Hypothesis()
	if len(hyp) > 0 {
		log.Printf("    > hypothesis: %s\n", hyp)
		l.textHypothesis = hyp
		l.speech, l.cpu, l.wall = l.dec.UttDuration()
		log.Printf("Utt duration: speech=%v, cpu=%v, wall=%v\n", l.speech, l.cpu, l.wall)
		allSpeech, allCpu, allWall := l.dec.AllDuration()
		log.Printf("All duration: speech=%v, cpu=%v, wall=%v\n", allSpeech, allCpu, allWall)

		// get lattice
		// DEBUGGING ONLY
		lat := l.dec.WordLattice()
		outhtkFileName := *l.infileName + ".htk"
		lat.WriteToHTK(sphinx.String(outhtkFileName))

		// get Segments

		pdec := l.dec.Decoder()
		seg := pocketsphinx.SegIter(pdec)
		// what to do ?
		var startFrame, endFrame, ascr, lscr, lback, pprob int32
		//TODO some intelligence about the size
		l.Response.Words = make([]Word, 0, 5)

		for ; seg != nil; seg = pocketsphinx.SegNext(seg) {
			word := pocketsphinx.SegWord(seg)
			pocketsphinx.SegFrames(seg, &startFrame, &endFrame)
			pprob = pocketsphinx.SegProb(seg, &ascr, &lscr, &lback)
			log.Printf("Seg word = %v, startFrame=%v, endFrame=%v, pprob=%v, ascr=%v, lscr=%v, lback=%v",
				word, startFrame, endFrame, pprob, ascr, lscr, lback)
			l.Response.Words = append(l.Response.Words,
				Word{Word: word,
					StartFrame: startFrame, EndFrame: endFrame,
					AScr: ascr, LScr: lscr,
					LBack: lback, PProb: pprob})
		}
		return
	}
	log.Println("NO RESULT??")
}
