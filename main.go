package main

import (
	"log"
	"os"
	"github.com/jawher/mow.cli"
	"github.com/xlab/closer"
	"github.com/xlab/pocketsphinx-go/sphinx"
	"github.com/quynhdang-vt/vt-pocketsphinx/cmu_sphinx"
)

const (
	samplesPerChannel = 512
	sampleRate        = 16000
	channels          = 1
)

var (
	app     = cli.App("qd-pocketsphinx", "Transcription engine using CMU pocketsphinx.")
	hmm     = app.StringOpt("hmm", "/usr/local/share/pocketsphinx/model/en-us/en-us", "Sets directory containing acoustic model files.")
	dict    = app.StringOpt("dict", "/usr/local/share/pocketsphinx/model/en-us/cmudict-en-us.dict", "Sets main pronunciation dictionary (lexicon) input file..")
	lm      = app.StringOpt("lm", "/usr/local/share/pocketsphinx/model/en-us/en-us.lm.bin", "Sets word trigram language model input file.")
	logfile = app.StringOpt("log", "/var/log/pocketsphinx.log", "Log file to write log to.")
	stdout  = app.BoolOpt("stdout", false, "Disables log file and writes everything to stdout.")
	infileName  = app.StringOpt("in", "test.wav", "wave file must be certain format")
	outraw  = app.StringOpt("outraw", "", "Specify output dir for RAW recorded sound files (s16le). Directory must exist.")
	payload = app.StringOpt("payload", "", "payload.json if invoked via veritone")
)



func main() {
	log.SetFlags(0)
	app.Action = appRun
	app.Run(os.Args)
}


/*
**
** for now, not yet handle the payload.json
*/
func appRun() {
	defer closer.Close()
	closer.Bind(func() {
		log.Println("Bye!")
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

    unitOfWork := &cmu_sphinx.UnitOfWork{}
    unitOfWork.InfileName = infileName
    unitOfWork.Dec = dec
    err = unitOfWork.Decode()
}



