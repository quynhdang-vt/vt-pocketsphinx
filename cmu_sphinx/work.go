package cmu_sphinx

import (
	"bufio"
	"encoding/json"
	"github.com/xlab/closer"
	"github.com/xlab/pocketsphinx-go/sphinx"
	qdUtils "github.com/quynhdang-vt/vt-pocketsphinx/utils"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
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
		log.Fatal("Invalid input file")
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

func (u *UnitOfWork) Decode() (ttml *string, transcriptDurationMs int, err error) {
	log.Println("UnitOfWork.Decode ENTER")
	defer log.Println("UnitOfWork.Decode EXIT")

	err = u.getFile()
	infileInfo, err := u.infile.Stat()
	qdUtils.CheckError(err)

	log.Printf("Processing %s, len=%d\n", infileInfo.Name(), infileInfo.Size())
	//    _, err = infile.Seek(44, 0)
	//    qdUtils.CheckError(err)

	var stream *bufio.Reader = bufio.NewReader(u.infile)

	in := make([]byte, 2048, 2048)

	l := &Recognizer{dec: u.Dec, infileName: u.InfileName}
	u.Dec.StartUtt()
	var totalframes int32
	for {
		nRead, err := stream.Read(in)
		//		log.Printf("Reading %d\n", nRead)

		if nRead == 0 || err == io.EOF {
			break
		}
		totalframes += l.Process(in, nRead)
	}
	u.Dec.EndUtt()
	l.Report() // report results

	var ttmlFileName string
	transcriptDurationMs = 0

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
		log.Println(">>>> LATTICE JSON")
		sJson := string(latticeJSON)
		//		log.Println(sJson)
		err = qdUtils.WriteToFile(*u.InfileName+".json", sJson)
		// upload ttml
		transcript := lattice.ToTranscript()
		s := transcript.ToTTML()
		ttml = &s
		log.Println(">>>> TTML")
		//		log.Println(ttml)
		ttmlFileName = *u.InfileName + ".ttml"
		u.TTMLFileName = &ttmlFileName
		err = qdUtils.WriteToFile(*u.InfileName+".ttml", *ttml)

		orderedLattice := lattice.ToOrderedLattice()

		if len(orderedLattice) != 0 {
			transcriptDurationMs = orderedLattice[len(orderedLattice)-1].StopTimeMs
		}

	}

	log.Printf("The END?.., # of frames = %v, transcriptDurationMs=%v \n", totalframes, transcriptDurationMs)
	return ttml, transcriptDurationMs, err
}

