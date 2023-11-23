# reconn - Self-hosted voice clone and inference service

reconn is a voice clone service based on
[Bark TTS](https://github.com/serp-ai/bark-with-voice-clone) that leverages
generative AI.

In addition, it uses a PostgreSQL database to model chat-bot like interaction
with cloned voice models, and distributes the intensive CUDA workload across
GPUs using an Azure service bus.

The program is only a short, incomplete programming exercise, notably missing
the integration with a chat bot service such as Microsoft bot framework and
Twilio.

## Components overview

-   A web server for the main app (aka "debug web app") running on CPU.
    *   Start `main.go` with `-debug=true -gpuworker=false`
    *   Start `ng serve` for interactive angular development, or `make all` to
        package both the angular app and go web server into a container image.
    *   The debug web app is a single page app and should be fairly self
        explanatory.
-   A voice clone and model inference server (aka "voicesvc") running on GPU.
    *   This requires a CUDA capable GPU and has only been tested on Windows.
    *   See `voicesvc/README.md` and `voicesvc/Pipfile` for dependency
        installation instructions.
-   A GPU worker server that picks up GPU tasks and forwards them to "voicesvc".
    *   This is a companion to "voicesvc" and typically started on the same
        GPU-equipped host, though the worker logic runs on CPU.
    *   Start `main.go` with `-debug=true -gpuworker=true`.

## Development tips

### Database server

Connect to the database using: `psql -h xx.postgres.database.azure.com -U
useruser -d postgres`.

List and create database:

```shell

postgres=> create database reconn;
CREATE DATABASE
postgres=> \l
                                                             List of databases
       Name        |     Owner      | Encoding |  Collate   |   Ctype    | ICU Locale | Locale Provider |         Access privileges
-------------------+----------------+----------+------------+------------+------------+-----------------+-----------------------------------
 azure_maintenance | azuresu        | UTF8     | en_US.utf8 | en_US.utf8 |            | libc            |
 azure_sys         | azuresu        | UTF8     | en_US.utf8 | en_US.utf8 |            | libc            |
 postgres          | azure_pg_admin | UTF8     | en_US.utf8 | en_US.utf8 |            | libc            |
 reconn            | reconnadmin    | UTF8     | en_US.utf8 | en_US.utf8 |            | libc            |
...
```

Use `\i` to load schema file and execute the DDL.

## Web server

### Start the backend server

Install go, then build and run the executable

```shell
# CWD: reconn repository root
go build
./reconn -port=8080
```

To run tests: `go test -v ./...`

### Start the frontend app with automated live reload

Install a couple of prerequisites:

```shell
# CWD: does not matter
# Install angular CLI
npm install -g @angular/cli

# Install chrome browser for test support
wget https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb
sudo apt install ./google-chrome-stable_current_amd64.deb
```

And then:

```shell
# CWD: reconn repository root
cd reconn-webapp
npm install # install angular dependencies
ng serve
```

Angular's auto-reloading web server will automatically proxy backend requests
`/api/*` to `localhost:8080`.

To run tests: `ng test --browsers ChromeHeadless`

### Build container image

The Makefile builds the backend (incl. web server) binary and frontend web app
assets, then copies both into a container image:

```shell
# CWD: reconn repository root
make
```

Note that the web app assets uses `ng build --base-href /resource/` to match the
expectation of the backend web server.

### Voice server

#### Start the voice server

Install pipenv (`pip install pipenv`), then enter pipenv shell, and follow the
instructions in `reconn/voicesvc/Pipfile` to install the dependencies.

To start on CUDA-capable Windows host:

```shell
PS C:\Users\guoho\Downloads\reconn\voicesvc> pipenv shell
Launching subshell in virtual environment...
...

PS C:\Users\guoho\Downloads\reconn\voicesvc> python main.py
2023-09-30 18:18:33 | INFO | root | about to start voice web service on 127.0.0.1:8081
2023-09-30 18:18:33 | INFO | root | using cuda for AI computing
2023-09-30 18:18:33 | INFO | root | using C:\tmp\voice_static_resource_dir for static resources
2023-09-30 18:18:33 | INFO | root | using C:\tmp\voice_sample_dir for voice sample storage
2023-09-30 18:18:33 | INFO | root | using C:\tmp\voice_model_dir for voice model storage
2023-09-30 18:18:33 | INFO | root | using C:\tmp\voice_temp_model_dir for temporary voice model storage
2023-09-30 18:18:33 | INFO | root | using C:\tmp\voice_output_dir for TTS output storage`
...
2023-09-30 18:31:32 | INFO | root | initialising flask url handlers
2023-09-30 18:31:32 | INFO | root | starting waitress wsgi web server on 127.0.0.1:8081
...
```

To run tests, stay in pipenv shell and run:

```shell
(voicesvc) ⋊> /m/c/U/g/D/r/voicesvc on main ⨯ python -m pytest
...
--------------------------------------------------------------------- live log setup ---------------------------------------------------------------------
INFO     root:svc.py:82 initialising flask url handlers
INFO     root:svc.py:91 initialising prerequisites
INFO     root:svc.py:92 using cpu for AI computing
...
```

## License

Copyright (C) 2023 Houzuo Guo All rights reserved.

This program is free software subject to the terms of GNU Public License, v 3.0.
You may find the license text in the LICENSE file.
