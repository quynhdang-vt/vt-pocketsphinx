package utils

import (
	"os/exec"
	"log"
	"strings"
	"time"
	"os"
	"fmt"
	"io/ioutil"
	"net/http"
	"io"
)
func NowInMs() int64 {
    return time.Now().UnixNano() / int64(time.Millisecond)
}
func GetFileInfo(filename string) (fileTypeInfo string, err error) {
	out, err := exec.Command("file", filename).Output()
	fileTypeInfo = string(out[:])
	log.Printf("GetFileInfo %v=%v\n", filename, fileTypeInfo)
	return fileTypeInfo, err
}

func GetTemporaryFileName (prefix, suffix string) string {
	return fmt.Sprintf("%s%v%s_%d.%s", os.TempDir(), os.PathSeparator, prefix, NowInMs(), suffix)
}

//eventually may need to do a list? for now assuming 1 
func ConvertFileToWave16KMono(infile string) (convertedFile string, err error){
	convertedFile=GetTemporaryFileName("ffmpeg", "wav")
	_, err = exec.Command("ffmpeg", 
	  	    "-hide_banner",
			"-i", infile,
			"-ar", "16000", // resample audio to 16000Hz 
			"-ac", "1", // convert to single-channel (mono)
			"-acodec", "pcm_s16le",
			convertedFile).Output()
	log.Printf("nConvertFileToWave16KMono Orig=%v, New=%v\n", infile, convertedFile)
	return convertedFile, err
}


// check on the file using the "file" command and see if it's a WAVE file,specifically
// RIFF (little-endian) data, WAVE audio, Microsoft PCM, 16 bit, mono
func  FileTypeSupported(filename string) (res bool, fileTypeInfo string) {
	const okWaveFile = "RIFF (little-endian) data, WAVE audio, Microsoft PCM, 16 bit, mono"
	fileTypeInfo, err := GetFileInfo(filename)
	CheckError(err)
	return strings.Contains(fileTypeInfo, okWaveFile), fileTypeInfo
}

var SuportedContentTypes = []string{"audio/wav", "audio/mpeg", "audio/mp3"}

func DownloadFile(url string, prefix string) (filepath string, err error) {

	log.Printf("DownloadFile ENTER url=%s --> %s\n", url, prefix)
	out, err := ioutil.TempFile("", prefix)
	if err != nil {
		return "", err
	}
	defer func() { out.Close(); log.Printf("DownloadFile EXIT") }()

	filepath = out.Name()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return filepath, nil
}
func IsSupportedContentType(contentType string) bool {
	for _, v := range SuportedContentTypes {
		if v == contentType {
			return true
		}
	}
	return false
}

func WriteToFile(fileName string, sdata string) (err error) {
	log.Printf("WriteToFile %s\n", fileName)
	f, err := os.Create(fileName)
	defer f.Close()
	CheckError(err)
	_, err = f.WriteString(sdata)
	return err
}

func CheckError(e error) {
	if e != nil {
		log.Fatal(e)
	}
}