// +build release autoupdate

package client

import (
	"fmt"
	update "github.com/inconshreveable/go-update"
	"github.com/inconshreveable/go-update/check"
	"ngrok/client/mvc"
	"ngrok/log"
	"ngrok/version"
	"time"
)

const appId = ""

var publicKey []byte = []byte(
	`-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0Gx8r9no1QBtCruJW2tu
082MJJ5ZA7k803GisR2c6WglPOD1b/+kUg+dx5Y0TKXz+uNlR3GrCxLh8WkoA95M
T38CQldIjoVN/bWP6jzFxL+6BRoKy5L1TcaIf3xb9B8OhwEq60cvFy7BBrLKEHJN
ua/D1S5axgNOAJ8tQ2w8gISICd84ng+U9tNMqIcEjUN89h3Z4zablfNIfVkbqbSR
fnkR9boUaMr6S1w8OeInjWdiab9sUr87GmEo/3tVxrHVCzHB8pzzoZceCkjgI551
d/hHfAl567YhlkQMNz8dawxBjQwCHHekgC8gAvTO7kmXkAm6YAbpa9kjwgnorPEP
ywIDAQAB
-----END PUBLIC KEY-----`)
var u *update.Update
var updateEndpoint = fmt.Sprintf("http://localhost:8889/1/Applications/%s/Update", appId)

func init() {
	var err error
	u, err = update.New().VerifySignatureWithPEM(publicKey)
	if err != nil {
		panic(err)
	}
}

func autoUpdate(s mvc.State, token string) {
	update := func() (tryAgain bool) {
		log.Info("Checking for update")
		params := check.Params{
			Version:    1,
			AppId:      appId,
			AppVersion: version.MajorMinor(),
			UserId:     token,
		}

		result, err := params.CheckForUpdate(updateEndpoint)
		if err == check.NoUpdateAvailable {
			log.Info("No update available")
			return true
		} else if err != nil {
			log.Error("Error while checking for update: %v", err)
			return true
		}

		if result.Initiative == check.INITIATIVE_AUTO {
			applyUpdate(s, result)
		} else if result.Initiative == check.INITIATIVE_MANUAL {
			// this is the way the server tells us to update manually
			log.Info("Server wants us to update manually")
			s.SetUpdateStatus(mvc.UpdateAvailable)
		} else {
			log.Info("Update available, but ignoring")
		}

		// stop trying after a single download attempt
		// XXX: improve this so the we can:
		// 1. safely update multiple times
		// 2. only retry after temporary errors
		return false
	}

	// try to update immediately and then at a set interval
	for {
		if tryAgain := update(); !tryAgain {
			break
		}

		time.Sleep(updateCheckInterval)
	}
}

func applyUpdate(s mvc.State, result *check.Result) {
	err, errRecover := result.Update(u)
	if err == nil {
		log.Info("Update ready!")
		s.SetUpdateStatus(mvc.UpdateReady)
		return
	}

	log.Error("Error while updating ngrok: %v", err)
	if errRecover != nil {
		log.Error("Error while recovering from failed ngrok update, your binary may be missing: %v", errRecover.Error())
	}

	// tell the user to update manually
	s.SetUpdateStatus(mvc.UpdateAvailable)
}
