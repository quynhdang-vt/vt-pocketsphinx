package cmu_sphinx

import (
	"encoding/binary"
	"github.com/xlab/pocketsphinx-go/pocketsphinx"
	"github.com/xlab/pocketsphinx-go/sphinx"
	"log"
	"time"
	"fmt"
)

// 1 every 10ms --> 10 per second
//
const NumOfFramesPerMs=1
type Recognizer struct {
	dec               *sphinx.Decoder
	infileName        *string
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

func (l *Recognizer) Process(raw []byte, nRaw int) (frames int32, err error) {
	frames, ok := l.dec.ProcessRaw(ConvertToInt16ArrayInLittleEndian(raw, nRaw), false, false)
//	log.Printf("processed: %d frames, ok: %v\n", frames, ok)
	if !ok {
		return 0, fmt.Errorf("Failed to process frames...")
	}
	return frames, nil
}



func (l *Recognizer) CollectData() {
	hyp, _ := l.dec.Hypothesis()
	if len(hyp) > 0 {
		log.Printf("    > hypothesis: %s\n", hyp)
		l.Response.Hypotheses = append(l.Response.Hypotheses, hyp)

		speech, cpu, wall := l.dec.UttDuration()
		l.speech += speech
		l.cpu += cpu
		l.wall += wall
		log.Printf("Utt duration: speech=%v, cpu=%v, wall=%v\n", l.speech, l.cpu, l.wall)
		allSpeech, allCpu, allWall := l.dec.AllDuration()
		log.Printf("All duration: speech=%v, cpu=%v, wall=%v\n", allSpeech, allCpu, allWall)

		// get lattice
		// DEBUGGING ONLY
		/*
			lat := l.dec.WordLattice()
			outhtkFileName := *l.infileName + ".htk"
			lat.WriteToHTK(sphinx.String(outhtkFileName))
		*/
		// get Segments

		pdec := l.dec.Decoder()
		seg := pocketsphinx.SegIter(pdec)
		// what to do ?
		var startFrame, endFrame, ascr, lscr, lback, pprob, startTime, endTime int32
		var confidence float64
		//TODO some intelligence about the size
		l.Response.Words = make([]Word, 0, 5)

		wordCount := 0
		for ; seg != nil; seg = pocketsphinx.SegNext(seg) {
			word := pocketsphinx.SegWord(seg)
			pocketsphinx.SegFrames(seg, &startFrame, &endFrame)
			pprob = pocketsphinx.SegProb(seg, &ascr, &lscr, &lback)
			confidence = pocketsphinx.LogmathExp(l.dec.LogMath().LogMath(), pprob)
			/*
				log.Printf("Seg word = %v, startFrame=%v, endFrame=%v, pprob=%v, ascr=%v, lscr=%v, lback=%v",
					word, startFrame, endFrame, pprob, ascr, lscr, lback)
			*/
			startTime = startFrame * NumOfFramesPerMs
			endTime = endFrame * NumOfFramesPerMs
			l.Response.Words = append(l.Response.Words,
				Word{Word: word,
					StartTime: startTime,
					EndTime: endTime,
					AScr: ascr, LScr: lscr,
					LBack: lback, PProb: pprob,
					Confidence: confidence})
			wordCount++
		}
		log.Printf("# of words in lattice: %d\n", wordCount)
		return
	}
	log.Println("NO RESULT??")
}
