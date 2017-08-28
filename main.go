package main

import (
    "io/ioutil"
    "encoding/json"
	"github.com/jawher/mow.cli"
	"github.com/quynhdang-vt/vt-pocketsphinx/cmu_sphinx"
	"github.com/quynhdang-vt/vt-pocketsphinx/models"
	veritoneAPI "github.com/veritone/go-veritone-api"
	"github.com/xlab/closer"
	"github.com/xlab/pocketsphinx-go/sphinx"
	"log"
	"os"
)

const (
	samplesPerChannel = 512
	sampleRate        = 16000
	channels          = 1
)

/**
PAYLOAD_FILE --> Payload JSON
API_URL: where to find API
API_TOKEN: authorization token to use on API requests

API_USERNAME: API username
API_PASSWORD: API password
*/
var (
	app        = cli.App("qd-pocketsphinx", "Transcription engine using CMU pocketsphinx.")
	hmm        = app.StringOpt("hmm", "/usr/local/share/pocketsphinx/model/en-us/en-us", "Sets directory containing acoustic model files.")
	dict       = app.StringOpt("dict", "/usr/local/share/pocketsphinx/model/en-us/cmudict-en-us.dict", "Sets main pronunciation dictionary (lexicon) input file..")
	lm         = app.StringOpt("lm", "/usr/local/share/pocketsphinx/model/en-us/en-us.lm.bin", "Sets word trigram language model input file.")
	logfile    = app.StringOpt("log", "/var/log/pocketsphinx.log", "Log file to write log to.")
	stdout     = app.BoolOpt("stdout", false, "Disables log file and writes everything to stdout.")
	infileName = app.StringOpt("in", "", "wave file must be certain format")
	outraw     = app.StringOpt("outraw", "/tmp", "Specify output dir for RAW recorded sound files (s16le). Directory must exist.")

	// as invoked from VDA
	payloadName = app.StringOpt("payload", os.Getenv("PAYLOAD_FILE"), "payload.json if invoked via veritone")
	apiToken    = app.StringOpt("apiToken", os.Getenv("API_TOKEN"), "API token")
	apiUrl      = app.StringOpt("apiUrl", os.Getenv("API_URL"), "API url")
	apiUsername = app.StringOpt("apiUsername", os.Getenv("API_USERNAME"), "API user name")
	apiPassword = app.StringOpt("apiPassword", os.Getenv("API_PASSWORD"), "API password")
)

// processPayload loads the payload file.
func getPayload(payloadName string) (p models.Payload) {
	// Try to load and marshal payload
	if len(payloadName) == 0 {
		log.Fatal("No payload provided")
	}
	payloadFile, err := ioutil.ReadFile(payloadName)
	if err != nil {
		log.Fatal("Unable to load payload file: " + err.Error())
	}

	if err = json.Unmarshal(payloadFile, &p); err != nil {
		log.Fatal("Error reading payload: " + err.Error())
	}

	log.Printf("Payload: %+v\n", p)
	return p
}

func main() {
	log.SetFlags(0)
	app.Action = appRun
	app.Run(os.Args)
}

func appRun() {
	defer closer.Close()
	closer.Bind(func() {
		log.Println("Bye!")
	})

	// Init CMUSphinx
	sphinxCfg := sphinx.NewConfig(
		sphinx.HMMDirOption(*hmm),
		sphinx.DictFileOption(*dict),
		sphinx.LMFileOption(*lm),
		sphinx.SampleRateOption(sampleRate),
	)
	if len(*outraw) > 0 {
		sphinx.RawLogDirOption(*outraw)(sphinxCfg)
	}
	if *stdout == false {
		sphinx.LogFileOption(*logfile)(sphinxCfg)
	}

	log.Println("Loading CMU PocketSphinx.")
	log.Println("This may take a while depending on the size of your model.")
	dec, err := sphinx.NewDecoder(sphinxCfg)
	if err != nil {
		closer.Fatalln(err)
	}
	closer.Bind(func() {
		dec.Destroy()
	})

	/** FOR TESTING ONLY */
	if len(*infileName) > 0 {
	    w := &cmu_sphinx.UnitOfWork{InfileName: infileName, Dec: dec}
	    w.Decode()
		os.Exit(0)
	}

	if len(*payloadName) == 0 ||
		(len(*apiToken) == 0 && len(*apiUsername) == 0 && len(*apiPassword) == 0) ||
		len(*apiUrl) == 0 {
		log.Fatal("Not given any context for engine to run??")
	}
	// may want to check the apiXXX variable as well.
	engineContext := models.EngineContext{
		APIToken:    apiToken,
		APIUrl:      apiUrl,
		APIUsername: apiUsername,
		APIPassword: apiPassword,
	}
	payload := getPayload(*payloadName)
	veritoneAPIConfig := veritoneAPI.APIConfig{
		Token:              *apiToken, // add token here
		BaseURI:            *apiUrl,   // Veritone API instance to use (dev/stage/etc.)
		Version:            "",       // API version to use
		MaxAttempts:        1,        // how many times to call Veritone API for each request until successful response
		Timeout:            "15s",    // API call timeout (for example: "3s")
		CreateAssetTimeout: "3m",     // CreateAsset API call timeout (for example: "1m")
		RetryDelay:         "15s",    // time to wait before each retry
	}
	// Create veritone api client
	veritoneAPIClient, err := veritoneAPI.New(veritoneAPIConfig)
	if err != nil {
		log.Fatal(err)
	}
	err = RunEngine(payload, engineContext, dec, veritoneAPIClient)
}
