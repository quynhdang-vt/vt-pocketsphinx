# Nuance Container

Container to run the Nuance transcription server and transcribe audio all in one.

## Engine Flow

### Build/Run Locally

Build:
```
make build-docker
```

Run Locally:
```
go get github.com/veritone/engine-sandbox
go install github.com/veritone/engine-sandbox
engine-sandbox -engineConfigFile conf/dev/config.json -enginePayloadFile payload.json -engineImage nuancetest -input PATH_TO_WAV_FILE.wav -outputDir ./
```

### What the golang binary does:

1) start golang binary
2) read payload/config (using iron lib)
3) generate nuance config.yaml using the language pack specified in the config (default to eng-us)
4) start NTE
5) run NTE
6) convert nuance lattice to VLF
7) upload assets
8) fin

### Deployment

#### Deploy Flow

1) Push changes to Github. Wercker build and docker-push steps will start
2) Deploy to iron.io from Wercker build step once docker-push step is complete

# Legacy Runtime Information

## Requirements to run:

You need a .netrc file to run and build this containing the github token to access the private git repos for Veritone.

## Wercker

The wercker file will get all of the necessary files and build the container on committing to the git repo. If you want to modify the container that is on iron you need to modify the wercker file that builds it. Note that this is totally separate from local builds if you use the Makefile/Dockerfile provided.
