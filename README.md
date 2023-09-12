# reconn

Be it POC, server-side, client-side - everything can belong here for now.

## Start the backend server

Install go, then build and run the executable

```shell
# CWD: reconn repository root
go build
./reconn -port=8080
```

To run tests: `go test -v ./...`

## Start the frontend app with automated live reload

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
