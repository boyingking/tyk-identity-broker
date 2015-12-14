package providers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/lonelycode/tyk-auth-proxy/tap"
	"github.com/lonelycode/tyk-auth-proxy/toth"
	"github.com/lonelycode/tyk-auth-proxy/tothic"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/gplus"
	"net/http"
	"strings"
)

var log = logrus.New()
var SocialLogTag = "[SOCIAL AUTH]"

type Social struct {
	handler tap.IdentityHandler
	config  GothConfig
	toth    toth.TothInstance
	profile tap.Profile
}

type GothProviderConfig struct {
	Name   string
	Key    string
	Secret string
}

type GothConfig struct {
	UseProviders []GothProviderConfig
}

func (s *Social) Name() string {
	return "SocialProvider"
}

func (s *Social) ProviderType() tap.ProviderType {
	return tap.REDIRECT_PROVIDER
}

func (s *Social) UseCallback() bool {
	return true
}

func (s *Social) Init(handler tap.IdentityHandler, profile tap.Profile, config []byte) error {
	s.handler = handler
	s.profile = profile

	s.toth = toth.TothInstance{}
	s.toth.Init()

	unmarshallErr := json.Unmarshal(config, &s.config)
	if unmarshallErr != nil {
		return unmarshallErr
	}

	gothProviders := []goth.Provider{}
	for _, provider := range s.config.UseProviders {
		switch provider.Name {
		case "gplus":
			gothProviders = append(gothProviders, gplus.New(provider.Key, provider.Secret, s.getCallBackURL(provider.Name)))
		}
	}

	s.toth.UseProviders(gothProviders...)
	return nil
}

func (s *Social) Handle(w http.ResponseWriter, r *http.Request) {
	tothic.BeginAuthHandler(w, r, &s.toth)
}

func (s *Social) checkConstraints(user interface{}) error {
	var thisUser goth.User
	thisUser = user.(goth.User)

	if s.profile.ProviderConstraints.Domain != "" {
		if !strings.HasSuffix(thisUser.Email, s.profile.ProviderConstraints.Domain) {
			return errors.New("Domain constraint failed, user domain does not match profile")
		}
	}

	if s.profile.ProviderConstraints.Group != "" {
		log.Warning("Social Auth does not support Group constraints")
	}

	return nil
}

func (s *Social) HandleCallback(w http.ResponseWriter, r *http.Request, onSuccess func(http.ResponseWriter, *http.Request, interface{}, tap.Profile), onError func(tag string, errorMsg string, rawErr error, code int, w http.ResponseWriter, r *http.Request)) {
	user, err := tothic.CompleteUserAuth(w, r, &s.toth)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	constraintErr := s.checkConstraints(user)
	if constraintErr != nil {
		onError(SocialLogTag, "Constraint failed", constraintErr, 400, w, r)
		return
	}

	onSuccess(w, r, user, s.profile)
}

func (s *Social) getCallBackURL(provider string) string {
	log.Warning("TODO: Callback URL must b dynamic!!!!")
	return "http://sharrow.tyk.io:3010/auth/" + s.profile.ID + "/" + provider + "/callback"
}
