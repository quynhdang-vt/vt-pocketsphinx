package main

import (
	"encoding/json"
	"fmt"
	"github.com/jawher/mow.cli"
	"github.com/quynhdang-vt/vt-pocketsphinx/cmu_sphinx"
	"github.com/quynhdang-vt/vt-pocketsphinx/models"
	veritoneAPI "github.com/veritone/go-veritone-api"
	"github.com/xlab/closer"
	"github.com/xlab/pocketsphinx-go/sphinx"
	"io/ioutil"
	"log"
	"net/url"
	"os"
)

const (
	samplesPerChannel = 512
	sampleRate        = 16000
	channels          = 1
)

/**
PAYLOAD_FILE --> Payload JSON
API_URL: where to find API  >> for now DEV URL is built into the Docker images

For initial development to get a token, will need the username, pw
API_USERNAME: API username
API_PASSWORD: API password
*/
var (
	app = cli.App("qd-pocketsphinx", "Transcription engine using CMU pocketsphinx.")

	// define your model with the hmm, dict, lm --> may need to get this somewhere eventually
	hmm  = app.StringOpt("hmm", "/usr/local/share/pocketsphinx/model/en-us/en-us", "Sets directory containing acoustic model files.")
	dict = app.StringOpt("dict", "/usr/local/share/pocketsphinx/model/en-us/cmudict-en-us.dict", "Sets main pronunciation dictionary (lexicon) input file..")
	lm   = app.StringOpt("lm", "/usr/local/share/pocketsphinx/model/en-us/en-us.lm.bin", "Sets word trigram language model input file.")

	logfile    = app.StringOpt("log", "/tmp/pocketsphinx.log", "Log file to write log to.")
	stdout     = app.BoolOpt("stdout", false, "Disables log file and writes everything to stdout.")
	infileName = app.StringOpt("in", "", "wave file must be certain format")
	outraw     = app.StringOpt("outraw", "/tmp", "Specify output dir for RAW recorded sound files (s16le). Directory must exist.")

	// as invoked from VDA
	payloadName = app.StringOpt("payload", os.Getenv("PAYLOAD_FILE"), "payload.json if invoked via veritone")

	// For local testing
	apiConfigFileName = app.StringOpt("apiConfigFileName", os.Getenv("API_CONFIG"), "configuration to geto to VTAPI")
	apiToken          = app.StringOpt("apiToken", os.Getenv("API_TOKEN"), "API token")
	apiUrl            = app.StringOpt("apiUrl", os.Getenv("API_URL"), "API url")
	apiUsername       = app.StringOpt("apiUsername", os.Getenv("API_USERNAME"), "API user name")
	apiPassword       = app.StringOpt("apiPassword", os.Getenv("API_PASSWORD"), "API password")
)

// processPayload loads the payload file.
func getPayload(payloadName string, apiConfigFileName string) (p models.Payload, e models.EngineContext) {
	// Try to load and marshal payload
	if len(payloadName) == 0 {
		log.Fatal("No payload provided")
	}
	payloadFile, err := ioutil.ReadFile(payloadName)
	log.Printf(">>>> READING PAYLOAD from %v\nFile Contents: %v\n", payloadName, string(payloadFile))
	if err != nil {
		log.Fatal("ERROR!!! Unable to load payload file: " + err.Error())
	}

	if err = json.Unmarshal(payloadFile, &p); err != nil {
		log.Fatal("Error reading payload: " + err.Error())
	}
	// see if token is there
	if len(p.Token) > 0 {
		e = models.EngineContext{APIToken: p.Token, APIUrl: os.Getenv("API_URL")}
	}
	if len(apiConfigFileName) > 0 {
		apiFileBuf, err := ioutil.ReadFile(apiConfigFileName)
		if err == nil {
			if err = json.Unmarshal(apiFileBuf, &e); err != nil {
				log.Printf("%v is not a JSON file, contents=%v\n", apiConfigFileName, apiFileBuf)
			}
		} else {
			log.Printf(">>>> ENGINE CONTEXT = %v\n", e)
		}
	}

	return p, e
}

func main() {
	log.SetFlags(0)
	app.Action = appRun
	app.Run(os.Args)
}

func parseAPIUrl(rawurl string) (baseURL string, pathNoSlash string) {
	url, err := url.Parse(rawurl)
	if err != nil {
		log.Fatalf("ERROR!!! parsing url=%v, err=%v", rawurl, err)
	}
	if len(url.Port()) > 0 {
		baseURL = fmt.Sprintf("%s://%s:%s", url.Scheme, url.Hostname(), url.Port())
	} else {
		baseURL = fmt.Sprintf("%s://%s", url.Scheme, url.Hostname())
	}
	if len(url.Path) > 1 {
		pathNoSlash = url.Path[1:len(url.Path)]
	}
	return baseURL, pathNoSlash
}
func appRun() {
	defer closer.Close()
	closer.Bind(func() {
		log.Println("Finished!")
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
	log.Println(">>>> Loading CMU PocketSphinx....This may take a while depending on the size of your model.")
	dec, err := sphinx.NewDecoder(sphinxCfg)
	if err != nil {
		closer.Fatalln(err)
	}
	closer.Bind(func() {
		dec.Destroy()
	})

	/** LOCAL TESTING ONLY */
	if len(*infileName) > 0 {
		w := &cmu_sphinx.UnitOfWork{InfileName: infileName, Dec: dec}
		ttml, latticeJson, interesting_tidbits, _ := w.Decode()
		log.Printf("TTML\n%s\n\n\nJSON\n%s\n\n\nINTERESTING\n%v\n", ttml, latticeJson, interesting_tidbits)
		os.Exit(0)
	}

	// START of the ENGINE now
	// the API_URL will be of this format: https://api.aws-dev.veritone.com/v1/
	// need to parse it and get the host:port since the go-veritone-api assumes base

	log.Println("--------- GETTING ENGINE info ------------")
	payload, engineContext := getPayload(*payloadName, *apiConfigFileName)
	// may want to check the apiXXX variable as well.
	if len(engineContext.APIToken) == 0 && len(*apiConfigFileName) == 0 {
		engineContext = models.EngineContext{
			APIToken:    *apiToken,
			APIUrl:      *apiUrl,
			APIUsername: *apiUsername, //not really in use.. but..
			APIPassword: *apiPassword,
		}
	}
	log.Printf("Payload: %+v\nEngine Context: %+v\n", payload, engineContext)

	// TODO need to get token if given username,pw for local dev effort

	if payload.IsInvalid() || engineContext.IsInvalid() {
		log.Fatal("Not given any context for engine to run??")
	}

	baseURL, version := parseAPIUrl(engineContext.APIUrl)
	veritoneAPIConfig := veritoneAPI.APIConfig{
		Token:              engineContext.APIToken, // add token here
		BaseURI:            baseURL,                // Veritone API instance to use (dev/stage/etc.)
		Version:            version,                // API version to use
		MaxAttempts:        1,                      // how many times to call Veritone API for each request until successful response
		Timeout:            "15s",                  // API call timeout (for example: "3s")
		CreateAssetTimeout: "3m",                   // CreateAsset API call timeout (for example: "1m")
		RetryDelay:         "15s",                  // time to wait before each retry
	}
	// Create veritone api client
	veritoneAPIClient, err := veritoneAPI.New(veritoneAPIConfig)
	if err != nil {
		log.Fatalf("ERROR!!! Failure to create an instance of Veritone API, err=%v\n", err)
	}
	err = RunEngine(payload, engineContext, dec, veritoneAPIClient)
	if err != nil {
		log.Printf("RunEngine got error? err=%v\n", err)
	}
}
