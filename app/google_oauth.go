// TODO(sadovsky): This should probably be a separate package.

package app

import (
	"encoding/json"
	"io"

	"code.google.com/p/goauth2/oauth"
)

const (
	SCOPE     = "https://www.googleapis.com/auth/userinfo.profile https://www.google.com/m8/feeds"
	AUTH_URL  = "https://accounts.google.com/o/oauth2/auth"
	TOKEN_URL = "https://accounts.google.com/o/oauth2/token"

	GOOGLE_API_REQUEST = "https://www.google.com/m8/feeds/contacts/default/full?max-results=10000&alt=json"
)

type idType struct {
	Email string `json:"$t"`
}

type titleType struct {
	Name string `json:"$t"`
}

type emailType struct {
	Address string
	Primary string
}

type entryType struct {
	Title  titleType
	Emails []emailType `json:"gd$email"`
}

type feedType struct {
	Id      idType
	Entries []entryType `json:"entry"`
}

type responseType struct {
	Feed feedType
}

////////////////////////////////////////
// Interface

type Contact struct {
	Name  string
	Email string
}

func GoogleParseContacts(r io.Reader) ([]*Contact, error) {
	var response responseType
	d := json.NewDecoder(r)
	if err := d.Decode(&response); err != nil {
		return nil, err
	}

	contacts := []*Contact{}
	for _, entry := range response.Feed.Entries {
		if entry.Title.Name == "" {
			continue
		}
		c := &Contact{Name: entry.Title.Name}
		for _, email := range entry.Emails {
			if email.Primary == "true" {
				c.Email = email.Address
				break
			}
		}
		if c.Email != "" {
			contacts = append(contacts, c)
		}
	}
	return contacts, nil
}

// NOTE(sadovsky): We set ApprovalPrompt to "force" so that we can easily
// recover from losing a refresh token.
func GoogleMakeConfig(tokenCache oauth.Cache) *oauth.Config {
	return &oauth.Config{
		ClientId:       kGoogleClientId,
		ClientSecret:   kGoogleClientSecret,
		Scope:          SCOPE,
		AuthURL:        AUTH_URL,
		TokenURL:       TOKEN_URL,
		RedirectURL:    kGoogleRedirectURL,
		TokenCache:     tokenCache,
		AccessType:     "offline",
		ApprovalPrompt: "force",
	}
}
