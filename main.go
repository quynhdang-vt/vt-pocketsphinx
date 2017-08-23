package main

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
    "bufio"
    "io"
    "strings"
    "time"
    "encoding/binary"
	"github.com/jawher/mow.cli"
	"github.com/xlab/closer"
	"github.com/xlab/pocketsphinx-go/sphinx"
	"github.com/xlab/pocketsphinx-go/pocketsphinx"	
	"github.com/quynhdang-vt/vt-pocketsphinx/cmu_sphinx"
)

const (
	samplesPerChannel = 512
	sampleRate        = 16000
	channels          = 1
)

var (
	app     = cli.App("qpocketsphinx", "TEST.")
	hmm     = app.StringOpt("hmm", "/usr/local/share/pocketsphinx/model/en-us/en-us", "Sets directory containing acoustic model files.")
	dict    = app.StringOpt("dict", "/usr/local/share/pocketsphinx/model/en-us/cmudict-en-us.dict", "Sets main pronunciation dictionary (lexicon) input file..")
	lm      = app.StringOpt("lm", "/usr/local/share/pocketsphinx/model/en-us/en-us.lm.bin", "Sets word trigram language model input file.")
	logfile = app.StringOpt("log", "gortana.log", "Log file to write log to.")
	stdout  = app.BoolOpt("stdout", false, "Disables log file and writes everything to stdout.")
	infileName  = app.StringOpt("in", "test.wav", "wave file must be certain format")
	outraw  = app.StringOpt("outraw", "", "Specify output dir for RAW recorded sound files (s16le). Directory must exist.")
)

func check(e error) {
	if e!=nil {
		log.Fatal(e)
	}
}


type Recognizer struct {
	inSpeech   bool
	uttStarted bool
	dec        *sphinx.Decoder
	textHypothesis       string
	speech, cpu, wall time.Duration
    Response   cmu_sphinx.Response
}

func (l *Recognizer) process( raw []byte, nRaw int ) (frames int32){
//	log.Println("process .. ", nRaw)
	// ProcessRaw with disabled search because callback needs to be relatime
	frames, ok := l.dec.ProcessRaw(convertToInt16ArrayInLittleEndian(raw, nRaw), false, false)
	// log.Printf("processed: %d frames, ok: %v", frames, ok)
	if !ok {
		log.Fatal("??? why is it not ok?")
	}
	return frames
}

func (l *Recognizer) report() {
	hyp, _ := l.dec.Hypothesis()
	if len(hyp) > 0 {
		log.Printf("    > hypothesis: %s\n", hyp)
		l.textHypothesis = hyp
	    l.speech, l.cpu, l.wall = l.dec.UttDuration()
     	log.Printf("speech=%v, cpu=%v, wall=%v\n", l.speech, l.cpu, l.wall)	
     	
     	// get lattice 
     	// DEBUGGING ONLY
     	lat := l.dec.WordLattice()
     	outhtkFileName := *infileName+".htk"
     	lat.WriteToHTK(sphinx.String(outhtkFileName))
     	
     	// get Segments
     	
     	pdec := l.dec.Decoder()
     	seg := pocketsphinx.SegIter(pdec)
     	// what to do ?
     	var startFrame, endFrame, ascr, lscr, lback, pprob int32
     	//TODO some intelligence about the size
     	l.Response.Words = make ([]cmu_sphinx.Word, 0, 5)

     	for ;seg != nil; seg = pocketsphinx.SegNext(seg) {
	     	word:= pocketsphinx.SegWord(seg)
     		pocketsphinx.SegFrames(seg, &startFrame, &endFrame)
     		pprob = pocketsphinx.SegProb(seg, &ascr, &lscr, &lback)
	     	log.Printf("Seg word = %v, startFrame=%v, endFrame=%v, pprob=%v, ascr=%v, lscr=%v, lback=%v",
	     		 word, startFrame, endFrame, pprob, ascr, lscr,lback)    
     		l.Response.Words = append( l.Response.Words,
     			 cmu_sphinx.Word{Word:word,
     			 	 StartFrame:startFrame, EndFrame:endFrame, 
     			 	 AScr:ascr, LScr:lscr, 
     			 	 LBack:lback, PProb:pprob}	)
     	}
		return
	}
	log.Println("NO RESULT??")
}
func main() {
	log.SetFlags(0)
	app.Action = appRun
	app.Run(os.Args)
}
// check on the file using the "file" command and see if it's a WAVE file,specifically
// RIFF (little-endian) data, WAVE audio, Microsoft PCM, 16 bit, mono
func checkIfValidFile (filename string) (res bool, fileTypeInfo string) {
	const okWaveFile = "RIFF (little-endian) data, WAVE audio, Microsoft PCM, 16 bit, mono"
	out, err := exec.Command("file", filename).Output()
	if err != nil {
		check(err)
	}
	fileTypeInfo = string(out[:])
	log.Printf("Checking for filetype=%s\n\n",fileTypeInfo)
	return strings.Contains(fileTypeInfo, okWaveFile), fileTypeInfo
}
func convertToInt16ArrayInLittleEndian( raw []byte, rawLen int) ([]int16) {
	const SIZEOF_INT16=2
	data := make([]int16, rawLen / SIZEOF_INT16)
	for i:=range data {
		data[i] = int16(binary.LittleEndian.Uint16(raw[i*SIZEOF_INT16:(i+1)*SIZEOF_INT16]))
	}
//	log.Printf("convertToInt16Array...from %d bytes to  %d int\n", rawLen, len(data))
	return data
}
func appRun() {
	defer closer.Close()
	closer.Bind(func() {
		log.Println("Bye!")
	})
	
	// TOODO may want to check if it's .wav etc..
	// read in the file 2K bytes at a time,
	// feed thru the decoder

    var fileTypeInfo string
    var fileOK bool
	if  fileOK, fileTypeInfo = checkIfValidFile(*infileName); !fileOK  {
		// TODO need to do something here.. specificall calling ffmpeg to do some conversion?
		log.Fatal("not sure about the file, it's not what I think it is ", fileTypeInfo)
	}
	log.Printf("%v\n",  fileTypeInfo)
	infile, err := os.Open(*infileName)

	closer.Bind(func() {
			if infile != nil {
			infile.Close()
			}
	})

	// Init CMUSphinx
	cfg := sphinx.NewConfig(
		sphinx.HMMDirOption(*hmm),
		sphinx.DictFileOption(*dict),
		sphinx.LMFileOption(*lm),
		sphinx.SampleRateOption(sampleRate),
	)
	if len(*outraw) > 0 {
		sphinx.RawLogDirOption(*outraw)(cfg)
	}
	if *stdout == false {
		sphinx.LogFileOption(*logfile)(cfg)
	}

	log.Println("Loading CMU PocketSphinx.")
	log.Println("This may take a while depending on the size of your model.")
	dec, err := sphinx.NewDecoder(cfg)
	if err != nil {
		closer.Fatalln(err)
	}
	closer.Bind(func() {
		dec.Destroy()
	})

	infileInfo, err:= infile.Stat()
	check(err)

	log.Printf("Processing %s, len=%d\n", infileInfo.Name(), infileInfo.Size())  
//    _, err = infile.Seek(44, 0)
//    check(err)
    
	var stream *bufio.Reader=bufio.NewReader(infile)
	
	in := make([]byte, 2048, 2048)

	l := &Recognizer{
		dec: dec,
	}	
	dec.StartUtt()
	var totalframes int32
	for {
		nRead, err := stream.Read(in)
//		log.Printf("Reading %d\n", nRead)
		
		if nRead==0 || err==io.EOF {
			break
		}
		totalframes += l.process(in, nRead)
	}
	dec.EndUtt()
	l.report() // report results
	// also this is where we got to do the stuff
	if l.Response.Words != nil {
		// see if we can convert to lattice
		lattice, err := l.Response.ToLattice()
		
		// then to file?
		// upload lattice
		latticeJSON, err := json.Marshal(&lattice)
		if err != nil {
			log.Fatalf("failed to marhsal lattice to json: %s", err)
		}
		// PRINT
		log.Println("LATTICE JSON")
		sJson := string(latticeJSON)
//		log.Println(sJson)
		err =WriteToFile(*infileName+".json", sJson)
		// upload ttml
		transcript := lattice.ToTranscript()
		ttml := transcript.ToTTML()
		log.Println("TTML")
//		log.Println(ttml)
		err =WriteToFile(*infileName+".ttml", ttml)
				
	}
	
	log.Println("The END?.., # of frames ", totalframes)

}

func WriteToFile (fileName string, sdata string) (err error) {
	log.Printf("Writing to %s\n", fileName)
	f, err := os.Create(fileName) 
	defer f.Close()
	check(err)
	_, err = f.WriteString(sdata)
	return err
}



