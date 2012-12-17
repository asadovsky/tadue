// Copyright 2012 Adam Sadovsky. All rights reserved.

// TODO(sadovsky): This should probably be a separate package.

package tadue

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

// TODO(sadovsky): Make these structs private.

type IdType struct {
	Email string `json:"$t"`
}

type TitleType struct {
	Name string `json:"$t"`
}

type EmailType struct {
	Address string
	Primary string
}

type EntryType struct {
	Title  TitleType
	Emails []EmailType `json:"gd$email"`
}

type FeedType struct {
	Id      IdType
	Entries []EntryType `json:"entry"`
}

type ResponseType struct {
	Feed FeedType
}

////////////////////////////////////////
// Interface

type Contact struct {
	Name  string
	Email string
}

func GoogleParseContacts(r io.Reader) ([]*Contact, error) {
	var response ResponseType
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
