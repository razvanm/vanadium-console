package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime/ppapi"
	"time"

	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"net/url"
	"runtime"
	"v.io/v23"
	"v.io/v23/context"
	"v.io/v23/logging"
	"v.io/v23/naming"
	"v.io/v23/security"
	"v.io/v23/vom"
	"v.io/x/ref/internal/logger"
	libsecurity "v.io/x/ref/lib/security"
	_ "v.io/x/ref/runtime/factories/chrome"
	"v.io/x/ref/runtime/protocols/lib/websocket"
)

const oauthBlesser = "https://dev.v.io/auth/google/bless"

type message struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

type instance struct {
	ppapi.Instance
	logger logging.Logger
	ctx    *context.T
}

var _ ppapi.InstanceHandlers = (*instance)(nil)

// crash prints an error and then panics. This is helpful because nacl doesn't
// seem to print a stack trace on panic (at least it doesn't by default in
// Chrome).
func crash(err error) {
	fmt.Print(err)
	panic(err)
}

func newInstance(inst ppapi.Instance) ppapi.InstanceHandlers {
	runtime.GOMAXPROCS(4)
	// Give the websocket interface the ppapi instance.
	websocket.PpapiInstance = inst
	ctx, _ := v23.Init()
	i := &instance{
		Instance: inst,
		logger:   logger.Global(),
		ctx:      ctx,
	}

	i.logger.Info("newInstance")
	return i
}

func (inst *instance) HandleMessage(messageVar ppapi.Var) {
	inst.logger.Infof("Got to HandleMessage(%+v)", messageVar)
	msgJSON, err := messageVar.AsString()
	fmt.Printf("message = %s err = %+v\n", msgJSON, err)
	if err != nil {
		inst.logger.Infof("Error: %+v", err)
		return
	}
	var msg message
	err = json.Unmarshal([]byte(msgJSON), &msg)
	if err != nil {
		inst.logger.Infof("Error: %+v", err)
		return
	}

	switch msg.Type {
	case "token":
		inst.blessings(msg.Data)
	case "url":
		// TODO(razvanm): issue a debug/http.RawDo.
		inst.glob("*")
	}
}

func (inst *instance) blessings(token string) {
	fmt.Printf("token = %q\n", token)

	principal := v23.GetPrincipal(inst.ctx)
	bytes, err := principal.PublicKey().MarshalBinary()
	if err != nil {
		crash(err)
	}
	expiry, err := security.NewExpiryCaveat(time.Now().Add(10 * time.Minute))
	if err != nil {
		crash(err)
	}
	caveats, err := base64VomEncode([]security.Caveat{expiry})
	if err != nil {
		crash(err)
	}
	// This interface is defined in:
	// https://godoc.org/v.io/x/ref/services/identity/internal/handlers#NewOAuthBlessingHandler
	v := url.Values{
		"public_key":    {base64.URLEncoding.EncodeToString(bytes)},
		"token":         {token},
		"caveats":       {caveats},
		"output_format": {"base64vom"},
	}
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			inst.ctx.Infof("retrying")
			time.Sleep(time.Second)
		}
		if body, err := inst.postBlessRequest(v); err == nil {
			var blessings security.Blessings
			if err := base64VomDecode(string(body), &blessings); err != nil {
				crash(err)
			}
			if err := libsecurity.SetDefaultBlessings(principal, blessings); err != nil {
				crash(err)
			}
			fmt.Println(principal.BlessingStore().DebugString())
			if err != nil {
				crash(err)
			}
			inst.glob("*")
			return
		} else {
			inst.ctx.Infof("error from oauth-blesser: %v", err)
		}
	}
}

func (inst *instance) glob(pattern string) {
	ctx, cancel := context.WithTimeout(inst.ctx, 5*time.Second)
	defer cancel()
	ns := v23.GetNamespace(ctx)
	fmt.Printf("roots: %v\n", ns.Roots())
	c, err := ns.Glob(ctx, pattern)

	if err != nil {
		crash(err)
	}

	for res := range c {
		switch v := res.(type) {
		case *naming.GlobReplyEntry:
			fmt.Fprintf(os.Stdout, "%s\n", v.Value.Name)
		case *naming.GlobReplyError:
			fmt.Fprintf(os.Stderr, "Error: name: %q value: %+v\n", v.Value.Name, v.Value.Error)
		}
	}
}

func (inst *instance) postBlessRequest(values url.Values) ([]byte, error) {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy:               func(req *http.Request) (*url.URL, error) { return nil, nil },
			Dial:                inst.Dial,
			TLSHandshakeTimeout: 10 * time.Second,
			TLSClientConfig: &tls.Config{
				RootCAs: newCertPool(),
			},
		},
	}
	resp, err := client.PostForm(oauthBlesser, values)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got %s", resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func newCertPool() *x509.CertPool {
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM([]byte(rootCerts))
	return certPool
}

func base64VomEncode(i interface{}) (string, error) {
	v, err := vom.Encode(i)
	return base64.URLEncoding.EncodeToString(v), err
}

func base64VomDecode(s string, i interface{}) error {
	b, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return err
	}
	return vom.Decode(b, i)
}

func main() {
	ppapi.Init(newInstance)
}

// The functions from below are just dummy implementations of the
// ppapi.InstanceHandlers interface.

func (inst *instance) DidCreate(args map[string]string) bool {
	inst.logger.Infof("Got to DidCreate")
	return true
}
func (inst *instance) DidDestroy() {
	inst.logger.Infof("Got to DidDestroy()")
}
func (inst *instance) DidChangeView(view ppapi.View) {
	inst.logger.Infof("Got to DidChangeView(%v)", view)
}
func (inst *instance) DidChangeFocus(has_focus bool) {
	inst.logger.Infof("Got to DidChangeFocus(%v)", has_focus)
}
func (inst *instance) HandleDocumentLoad(url_loader ppapi.Resource) bool {
	inst.logger.Infof("Got to HandleDocumentLoad(%v)", url_loader)
	return true
}
func (inst *instance) HandleInputEvent(event ppapi.InputEvent) bool {
	inst.logger.Infof("Got to HandleInputEvent(%v)", event)
	return true
}
func (inst *instance) Graphics3DContextLost() {
	inst.logger.Infof("Got to Graphics3DContextLost()")
}
func (inst *instance) MouseLockLost() {
	inst.logger.Infof("Got to MouseLockLost()")
}

// The current root for v.io is the GoDaddy's root CA cert that expires in 2037.
const rootCerts = `-----BEGIN CERTIFICATE-----
MIIDxTCCAq2gAwIBAgIBADANBgkqhkiG9w0BAQsFADCBgzELMAkGA1UEBhMCVVMx
EDAOBgNVBAgTB0FyaXpvbmExEzARBgNVBAcTClNjb3R0c2RhbGUxGjAYBgNVBAoT
EUdvRGFkZHkuY29tLCBJbmMuMTEwLwYDVQQDEyhHbyBEYWRkeSBSb290IENlcnRp
ZmljYXRlIEF1dGhvcml0eSAtIEcyMB4XDTA5MDkwMTAwMDAwMFoXDTM3MTIzMTIz
NTk1OVowgYMxCzAJBgNVBAYTAlVTMRAwDgYDVQQIEwdBcml6b25hMRMwEQYDVQQH
EwpTY290dHNkYWxlMRowGAYDVQQKExFHb0RhZGR5LmNvbSwgSW5jLjExMC8GA1UE
AxMoR28gRGFkZHkgUm9vdCBDZXJ0aWZpY2F0ZSBBdXRob3JpdHkgLSBHMjCCASIw
DQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAL9xYgjx+lk09xvJGKP3gElY6SKD
E6bFIEMBO4Tx5oVJnyfq9oQbTqC023CYxzIBsQU+B07u9PpPL1kwIuerGVZr4oAH
/PMWdYA5UXvl+TW2dE6pjYIT5LY/qQOD+qK+ihVqf94Lw7YZFAXK6sOoBJQ7Rnwy
DfMAZiLIjWltNowRGLfTshxgtDj6AozO091GB94KPutdfMh8+7ArU6SSYmlRJQVh
GkSBjCypQ5Yj36w6gZoOKcUcqeldHraenjAKOc7xiID7S13MMuyFYkMlNAJWJwGR
tDtwKj9useiciAF9n9T521NtYJ2/LOdYq7hfRvzOxBsDPAnrSTFcaUaz4EcCAwEA
AaNCMEAwDwYDVR0TAQH/BAUwAwEB/zAOBgNVHQ8BAf8EBAMCAQYwHQYDVR0OBBYE
FDqahQcQZyi27/a9BUFuIMGU2g/eMA0GCSqGSIb3DQEBCwUAA4IBAQCZ21151fmX
WWcDYfF+OwYxdS2hII5PZYe096acvNjpL9DbWu7PdIxztDhC2gV7+AJ1uP2lsdeu
9tfeE8tTEH6KRtGX+rcuKxGrkLAngPnon1rpN5+r5N9ss4UXnT3ZJE95kTXWXwTr
gIOrmgIttRD02JDHBHNA7XIloKmf7J6raBKZV8aPEjoJpL1E/QYVN8Gb5DKj7Tjo
2GTzLH4U/ALqn83/B2gX2yKQOC16jdFU8WnjXzPKej17CuPKf1855eJ1usV2GDPO
LPAvTK33sefOT6jEm0pUBsV/fdUID+Ic/n4XuKxe9tQWskMJDE32p2u0mYRlynqI
4uJEvlz36hz1
-----END CERTIFICATE-----`