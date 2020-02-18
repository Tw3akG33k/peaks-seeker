package provider

import (
	"context"
	"time"
	"sync"
	"strings"

	"go.uber.org/zap"
	log	"github.com/sirupsen/logrus"
	"github.com/google/go-github/v29/github"
	"github.com/lucmichalski/peaks-seeker/pkg/data"
	"github.com/lucmichalski/peaks-seeker/pkg/client"
)

// GetUserInfo fills in the given User struct with all available data
func GetUserInfo(ghc *client.GHClient, ghcm *client.ClientManager, user *data.User) error {

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	var (
		// client *ghclient.GHClient
		ok      bool
		wg      sync.WaitGroup
		res 	*github.User
		e       *github.AbuseRateLimitError
	)

getUser:
	res, resp, err := ghc.Client.Users.Get(ctx, user.Username)
	checkForRemainingLimit(ghc, ghcm, true, 5, true)
	log.Infoln(resp.Request.URL.String())

	if err != nil {
		if _, ok = err.(*github.RateLimitError); ok {
			log.Error("GetUserInfo hit limit error, it's time to change client.", zap.Error(err))

			goto changeClient
		} else if e, ok = err.(*github.AbuseRateLimitError); ok {
			log.Error("GetUserInfo have triggered an abuse detection mechanism.", zap.Error(err))
			time.Sleep(*e.RetryAfter)
			goto getUser

		} else if strings.Contains(err.Error(), "timeout") {
			log.Info("GetUserInfo has encountered a timeout error. Sleep for five minutes.")
			time.Sleep(5 * time.Minute)
			goto getUser

		} else {
			log.Error("GetUserInfo terminated because of this error.", zap.Error(err))
			return err
		}
	}

	user.Name = ptrString(res.Name)
	user.Location = ptrString(res.Location)
	user.Company = ptrString(res.Company)
	user.Email = ptrString(res.Email)
	user.Blog = ptrString(res.Blog)
	user.Hireable = ptrBool(res.Hireable)
	user.Bio = ptrString(res.Bio)
	user.AvatarURL = ptrString(res.AvatarURL)
	user.HTMLURL = ptrString(res.HTMLURL)

changeClient:
	{
		log.Warnln("GetUserInfo.changeClient...")
		go func() {
			wg.Add(1)
			defer wg.Done()
			client.Reclaim(ghc, resp)
		}()
		ghc = ghcm.Fetch()
		goto getUser
	}

	return nil

}
