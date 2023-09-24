# reconn

Be it POC, server-side, client-side - everything can belong here for now.

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

Install pipenv (`pip install pipenv`) and then activate pip environment and run:

```shell
⋊> /m/c/U/g/D/r/voicesvc on main ⨯ pipenv shell
Launching subshell in virtual environment...
 source /home/howard/.local/share/virtualenvs/voicesvc-gxZigfdY/bin/activate.fish
⋊> /m/c/U/g/D/r/voicesvc on main ⨯  source /home/howard/.local/share/virtualenvs/voicesvc-gxZigfdY/bin/activate.fish

(voicesvc) ⋊> /m/c/U/g/D/r/voicesvc on main ⨯ ./main.py
 * Serving Flask app 'app'
 * Debug mode: off
WARNING: This is a development server. Do not use it in a production deployment. Use a production WSGI server instead.
 * Running on http://127.0.0.1:8081
Press CTRL+C to quit
...
```

To run tests, activate pip environment and run:

```shell
(voicesvc) ⋊> /m/c/U/g/D/r/voicesvc on main ⨯ pytest
================================================= test session starts ==================================================
platform linux -- Python 3.11.4, pytest-7.4.2, pluggy-1.3.0
rootdir: /mnt/c/Users/guoho/Downloads/reconn/voicesvc
...
```
