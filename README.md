# reconn

Be it POC, server-side, client-side - everything can belong here for now.

## Database server

Connect to the database using: `psql -h reconn-user-db.postgres.database.azure.com -U reconnadmin -d postgres`, the database password `BOInscINOnioVc2RK`.

List and create database:

``` shell

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

Angular's auto-reloading web server will automatically proxy backend requests `/api/*` to `localhost:8080`.

To run tests: `ng test --browsers ChromeHeadless`

### Build container image

The Makefile builds the backend (incl. web server) binary and frontend web app assets, then copies both into a container image:

```shell
# CWD: reconn repository root
make
```

Note that the web app assets uses `ng build --base-href /resource/` to match the expectation of the backend web server.

## Voice server

### Start the voice server

Install pipenv (`pip install pipenv`), then enter pipenv shell, and follow the instructions in `reconn/voicesvc/Pipfile` to install the dependencies.

Start the development web server using `./main.py --debug --ai_computing_device=cpu` (too slow for AI inference):

```shell
⋊> /m/c/U/g/D/r/voicesvc on main ⨯ pipenv shell
Launching subshell in virtual environment...
 source /home/howard/.local/share/virtualenvs/voicesvc-gxZigfdY/bin/activate.fish
⋊> /m/c/U/g/D/r/voicesvc on main ⨯  source /home/howard/.local/share/virtualenvs/voicesvc-gxZigfdY/bin/activate.fish

(voicesvc) ⋊> /m/c/U/g/D/r/voicesvc on main ⨯ ./main.py --debug --ai_computing_device=cpu
2023-09-30 18:06:18 | INFO | root | about to start voice web service on 127.0.0.1:8081
2023-09-30 18:06:18 | INFO | root | using cpu for AI computing
2023-09-30 18:06:18 | INFO | root | using /tmp/voice_static_resource_dir for static resources
2023-09-30 18:06:18 | INFO | root | using /tmp/voice_sample_dir for voice sample storage
2023-09-30 18:06:18 | INFO | root | using /tmp/voice_model_dir for voice model storage
2023-09-30 18:06:18 | INFO | root | using /tmp/voice_temp_model_dir for temporary voice model storage
2023-09-30 18:06:18 | INFO | root | using /tmp/voice_output_dir for TTS output storage
...
 * Serving Flask app 'app'
 * Debug mode: off
WARNING: This is a development server. Do not use it in a production deployment. Use a production WSGI server instead.
 * Running on http://127.0.0.1:8081
Press CTRL+C to quit
...
```

To start on CUDA-capable Windows host:

``` shell
PS C:\Users\guoho\Downloads\reconn\voicesvc> pipenv shell
Launching subshell in virtual environment...
Windows PowerShell
Copyright (C) Microsoft Corporation. All rights reserved.

Install the latest PowerShell for new features and improvements! https://aka.ms/PSWindows

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
