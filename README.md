# App-Attest-Middleware
![](https://img.shields.io/badge/go-%3E%3D%201.11-blue)

Golang's HTTP middleware package that handles assertions and assertion requests for Apple apps and supports server-side validation.

[App-Attest](https://github.com/takimoto3/app-attest) package is used for verification.

## System Requirements

* Go 1.11 (or newer)


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
    attestHandler := handler.AttestationHandler{
		Logger: &logger.StdLogger{},
		AttestationService: &attest.AttestationService{
			AppID:         "<TEAM ID>.<Bundle ID>", // AppID
			PathForRootCA: "certs/Apple_App_Attestation_Root_CA.pem",
		},
		AttestationPlugin: &MyAttestationPlugin{}, // Your attestation Plugin
    }

    assert := middleware.AppAttestAssert (
        &logger.StdLogger{},
        "<TEAM ID>.<Bundle ID>", // AppID
        &MyAssertPlugin{....},  // Your assert Plugin
    )

    http.HandleFunc("/attest/", attestHandler.NewChallenge)
    http.HandleFunc("/attest/add", attestHandler.VerifyAttestation)
    http.HandleFunc("/hello", assert(Hello))

    err := http.ListenAndServe("0.0.0.0:3000", nil)
}

func Hello(w http.ResponseWriter, r *http.Request) {}
```

### Testing

To run the test, you need to prepare the test data(JSON). For information on how to create test data, see Testing in the [App-Attest](https://github.com/takimoto3/app-attest) package.

 * testdata/attestation.json


### Sample

Please refer to the sample web application that uses this package [here](https://github.com/takimoto3/app-attest-sample).

## License

App-Attest-Middleware is available under the MIT license. See the LICENSE file for more info.