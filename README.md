# App-Attest-Middleware
![](https://img.shields.io/badge/go-%3E%3D%201.16-blue)

Golang's HTTP middleware package that handles assertions and assertion requests for Apple apps and supports server-side validation.

[App-Attest](https://github.com/takimoto3/app-attest) package is used for verification.

## System Requirements

* Go 1.16 (or newer)


## Installation

```sh
# go 1.16 and newer versions. 
go install github.com/takimoto3/app-attest-middleware
```
```sh
# other
go get -u github.com/takimoto3/app-attest-middleware
```


## Usage
It consists of http middleware and handlers, but it doesn't work on its own. To run, you need to implement the Attestation and Assertion plugins, respectively.

#### AttestationPlugin
```go
type AttestationPlugin interface {
	// GetAssignedChallenge returns the assigned challenge.
	GetAssignedChallenge(r *http.Request) (string, error)
	// ResponseNewChallenge sets the newly generated challenge to the http response.
	ResponseNewChallenge(w http.ResponseWriter, r *http.Request) error
	// ParseRequest returns http.Request, AttstationObject, the hash value of challenge, and KeyID from the argument http.Request.
	ParseRequest(r *http.Request) (*http.Request, *attest.AttestationObject, []byte, []byte, error)
	// StoreResult stores the data of the argument attest.Result (PublicKey, Receipt, etc.).
	StoreResult(r *http.Request, result *attest.Result) error
}
```

#### AttestationPlugin
```go
type AssertionPlugin interface {
	// GetAssignedChallenge returns the assigned challenge.
	GetAssignedChallenge(r *http.Request) (string, error)
	// ResponseNewChallenge sets the newly generated challenge to the http response.
	ResponseNewChallenge(w http.ResponseWriter, r *http.Request) error
	// ParseRequest returns http.Request, AssertionObject and challenge from the arguments.
	ParseRequest(r *http.Request, requestBody []byte) (*http.Request, *attest.AssertionObject, string, error)
	// RedirectToAttestation redirects to the URL where you want to save the attestation.
	RedirectToAttestation(w http.ResponseWriter, r *http.Request)
	// GetPublicKeyAndCounter returns the stored public key and counter.
	GetPublicKeyAndCounter(r *http.Request) (*ecdsa.PublicKey, uint32, error)
	// StoreNewCounter save the assertion counter.
	StoreNewCounter(r *http.Request, counter uint32) error
}
```
Once you have a structure that implements the above plugin interface, use it as follows:
##### Setup Handler And Middleware

```go
func main() {
    // Note: MyAttestationPlugin and MyAssertionPlugin must be implemented by the user.
    attestationPlugin := &MyAttestationPlugin{}
    assertionPlugin := &MyAssertionPlugin{}

    attestHandler := handler.AttestationHandler{
		Logger: &logger.StdLogger{Logger: log.New(os.Stdout, "", 0)},
		AttestationService: &attest.AttestationService{
			AppID:         "<TEAM ID>.<Bundle ID>",
			PathForRootCA: "certs/Apple_App_Attestation_Root_CA.pem", 
		},
		AttestationPlugin: attestationPlugin,
    }

    assertionMiddleware := middleware.NewMiddleware(
        &logger.StdLogger{Logger: log.New(os.Stdout, "", 0)},
        "<TEAM ID>.<Bundle ID>",
        assertionPlugin,
    )

    helloHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

    http.HandleFunc("/attest/challenge", attestHandler.NewChallenge)
    http.HandleFunc("/attest/verify", attestHandler.VerifyAttestation)
    http.Handle("/hello", assertionMiddleware.AttestAssertWith(helloHandler))

    log.Println("Listening on :8080...")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```


## License

App-Attest-Middleware is available under the MIT license. See the LICENSE file for more info.