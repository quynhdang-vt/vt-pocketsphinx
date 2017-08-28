package cmu_sphinx

import (
	"encoding/json"
	"log"
	"io"
	"os"
	"os/exec"
	"bufio"
    "strings"	
	"github.com/xlab/closer"	
	"github.com/xlab/pocketsphinx-go/sphinx"
)

type UnitOfWork struct {
	InfileName *string
	Dec        *sphinx.Decoder
	results    map[string]string
	infile     *os.File
	TTMLFileName *string
}

// check on the file using the "file" command and see if it's a WAVE file,specifically
// RIFF (little-endian) data, WAVE audio, Microsoft PCM, 16 bit, mono
func (u* UnitOfWork) ValidateInput () (res bool, fileTypeInfo string) {
	const okWaveFile = "RIFF (little-endian) data, WAVE audio, Microsoft PCM, 16 bit, mono"
	out, err := exec.Command("file", *u.InfileName).Output()
	if err != nil {
		check(err)
	}
	fileTypeInfo = string(out[:])
	log.Printf("Checking for filetype=%s\n\n",fileTypeInfo)
	return strings.Contains(fileTypeInfo, okWaveFile), fileTypeInfo
}

func (u* UnitOfWork) getFile()(err error) {
    var fileTypeInfo string
    var fileOK bool
	if  fileOK, fileTypeInfo = u.ValidateInput(); !fileOK  {
		// TODO need to do something here.. specifically calling ffmpeg to do some conversion?
		log.Fatal("Invalid input file -- ", fileTypeInfo)
	}
	log.Println(fileTypeInfo)
	infile, err := os.Open(*u.InfileName)
	u.infile = infile

	closer.Bind(func() {
			if u.infile != nil {
			u.infile.Close()
			}
	})
	return err
}

func (u* UnitOfWork) Decode ()(ttml *string , transcriptDurationMs int,  err error) {
    err = u.getFile()
	infileInfo, err:= u.infile.Stat()
	check(err)

	log.Printf("Processing %s, len=%d\n", infileInfo.Name(), infileInfo.Size())  
//    _, err = infile.Seek(44, 0)
//    check(err)
    
	var stream *bufio.Reader=bufio.NewReader(u.infile)
	
	in := make([]byte, 2048, 2048)

	l := &Recognizer{dec:u.Dec, infileName:u.InfileName}	
	u.Dec.StartUtt()
	var totalframes int32
	for {
		nRead, err := stream.Read(in)
//		log.Printf("Reading %d\n", nRead)
		
		if nRead==0 || err==io.EOF {
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
		log.Println("LATTICE JSON")
		sJson := string(latticeJSON)
//		log.Println(sJson)
		err =WriteToFile(*u.InfileName+".json", sJson)
		// upload ttml
		transcript := lattice.ToTranscript()
		s := transcript.ToTTML()
		ttml = &s
		log.Println("TTML")
//		log.Println(ttml)
        ttmlFileName = *u.InfileName+".ttml"
        u.TTMLFileName = &ttmlFileName
		err =WriteToFile(*u.InfileName+".ttml", *ttml)
		
			orderedLattice := lattice.ToOrderedLattice()

			if len(orderedLattice) != 0 {
				transcriptDurationMs = orderedLattice[len(orderedLattice)-1].StopTimeMs
			}
						
	}
	
	log.Println("The END?.., # of frames ", totalframes)
	return ttml, transcriptDurationMs, err
}


func WriteToFile (fileName string, sdata string) (err error) {
	log.Printf("Writing to %s\n", fileName)
	f, err := os.Create(fileName) 
	defer f.Close()
	check(err)
	_, err = f.WriteString(sdata)
	return err
}



func check(e error) {
	if e!=nil {
		log.Fatal(e)
	}
}