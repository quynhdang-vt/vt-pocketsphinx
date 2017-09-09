package cmu_sphinx

import (
	"bufio"
	"encoding/json"
	qdUtils "github.com/quynhdang-vt/vt-pocketsphinx/utils"
	"github.com/xlab/closer"
	"github.com/xlab/pocketsphinx-go/sphinx"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"fmt"
)

type UnitOfWork struct {
	InfileName   *string
	Dec          *sphinx.Decoder
	results      map[string]string
	infile       *os.File
	TTMLFileName *string
}

func (u *UnitOfWork) getFile() (err error) {
	var fileTypeInfo string
	var fileOK bool
	if fileOK, fileTypeInfo = qdUtils.FileTypeSupported(*u.InfileName); !fileOK {
		// TODO need to do something here.. specifically calling ffmpeg to do some conversion?
		// if ASCII then we may want to dump it out for debugging purpose?
		buf := []byte{}
		if strings.Contains(fileTypeInfo, "ASCII") {
			buf, err = ioutil.ReadFile(*u.InfileName)
			log.Printf("unsupported ASCII: >>>%v<<<\n", buf)
		}
		return fmt.Errorf("Invalid input file format")
	}
	infile, err := os.Open(*u.InfileName)
	u.infile = infile

	closer.Bind(func() {
		if u.infile != nil {
			u.infile.Close()
		}
	})
	return err
}

func (u *UnitOfWork) Decode() (ttml []byte, latticeJson []byte, interesting_tidbits map[string]string, err error) {
	log.Println("UnitOfWork.Decode ENTER")
	defer log.Println("UnitOfWork.Decode EXIT")

	err = u.getFile()
	if err!=nil {
		return nil, nil, nil, err
	}
	_, err = u.infile.Stat()
	if err!=nil {
		return nil, nil, nil, err
	}

//	log.Printf("Processing %s, len=%d\n", infileInfo.Name(), infileInfo.Size())

	var stream *bufio.Reader = bufio.NewReader(u.infile)

	in := make([]byte, 2048, 2048)

	l := &Recognizer{dec: u.Dec, infileName: u.InfileName}
	if (!u.Dec.StartUtt()) {
		return nil, nil, nil, fmt.Errorf("Failed to start processing...")
	}
	utt_started := false

	var totalFrames, nFrames int32
	for {
		nRead, err := stream.Read(in)
		if nRead == 0 || err == io.EOF {
			break
		}
		nFrames, err = l.Process(in, nRead)
		totalFrames+=nFrames
		if ( u.Dec.IsInSpeech() ){
			utt_started = true
		} else if utt_started {
			u.Dec.EndUtt()
			utt_started=false
			l.CollectData()
			u.Dec.StartUtt()
		}
	}
	// wrapped up
	u.Dec.EndUtt()
	if (utt_started) {
		l.CollectData()
	}

	interesting_tidbits = make(map[string]string)


	if l.Response.Words != nil {
		// see if we can convert to lattice
		lattice, err := l.Response.ToLattice()
		if err == nil {

			// Veritone Lattice to JSON
			latticeJson, err = json.Marshal(&lattice)
			if err != nil {
				log.Fatalf("failed to marhsal lattice to json: %s", err)
			}
			/** DEBUG
			log.Println(">>>> LATTICE JSON")

			ss := string(latticeJSON)
			qdUtils.WriteToFile(*u.InfileName+".json", ss)
			*/

			// Get TTML, this depends on a specific go-lattice version
			transcript := lattice.ToTranscript()
			s := transcript.ToTTML()
			ttml = []byte(s)
			/*
			log.Println(">>>> TTML")

			u.TTMLFileName = &ttmlFileName
			err = qdUtils.WriteToFile(*u.InfileName+".ttml", *ttml)
			*/
			interesting_tidbits["speech_time"] = fmt.Sprintf("%v", l.speech)
			interesting_tidbits["cpu_time"] = fmt.Sprintf("%v", l.cpu)
			interesting_tidbits["total_frames"] = fmt.Sprintf("%v", totalFrames)
		}
	}

	return ttml, latticeJson, interesting_tidbits, err
}
