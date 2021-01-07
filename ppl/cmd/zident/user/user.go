package user

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/brimsec/zq/ppl/cmd/zident/root"
	"github.com/go-resty/resty/v2"
	"github.com/mccanne/charm"
	"github.com/segmentio/ksuid"
	"go.uber.org/multierr"
	"gopkg.in/auth0.v5/management"
)

const auth0DefaultConnection = "Username-Password-Authentication"

type auth0ClientConfig struct {
	clientId     string
	clientSecret string
	connection   string
	domain       url.URL
}

func (c *auth0ClientConfig) FromEnv() error {
	var err error
	if c.clientId = os.Getenv("AUTH0_CLIENT_ID"); c.clientId == "" {
		err = multierr.Append(err, errors.New("AUTH0_CLIENT_ID not set"))
	}
	if c.clientSecret = os.Getenv("AUTH0_CLIENT_SECRET"); c.clientSecret == "" {
		err = multierr.Append(err, errors.New("AUTH0_CLIENT_SECRET not set"))
	}
	if c.connection = os.Getenv("AUTH0_CONNECTION"); c.connection == "" {
		err = multierr.Append(err, errors.New("AUTH0_CONNECTION not set"))
	}
	domain := os.Getenv("AUTH0_DOMAIN")
	if domain == "" {
		err = multierr.Append(err, errors.New("AUTH0_DOMAIN not set"))
	} else {
		u, err := url.Parse(domain)
		if err != nil {
			err = multierr.Append(err, errors.New("AUTH0_DOMAIN is not a URL"))
		} else {
			c.domain = *u
		}
	}
	if err != nil {
		return fmt.Errorf("unable to configure auth0 client: %w", err)
	}
	return nil
}

type searchFlags struct {
	email    string
	tenantID string
	userID   string
}

func (f *searchFlags) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&f.email, "email", "", "search by email address")
	fs.StringVar(&f.tenantID, "tenantid", "", "search by tenant ID")
	fs.StringVar(&f.userID, "userid", "", "search by user ID")
}

func findUser(ctx context.Context, m *management.Management, f searchFlags) (*management.User, error) {
	var user *management.User
	if err := streamUsers(ctx, m, f, false, func(u *management.User) error {
		user = u
		return nil
	}); err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("no user found")
	}
	return user, nil
}

func streamUsers(ctx context.Context, m *management.Management, f searchFlags, allowMultiple bool, fn func(*management.User) error) error {
	// Auth0 user search syntax documented here:
	// https://auth0.com/docs/users/user-search/user-search-query-syntax
	var sstr []string
	if f.email != "" {
		sstr = append(sstr, fmt.Sprintf(`email:%s`, f.email))
	}
	if f.userID != "" {
		sstr = append(sstr, fmt.Sprintf(`app_metadata.brim_user_id:%s`, f.userID))
	}
	if f.tenantID != "" {
		sstr = append(sstr, fmt.Sprintf(`app_metadata.brim_tenant_id:%s`, f.tenantID))
	}
	query := management.Query(strings.Join(sstr, " AND "))

	var page int
	for {
		res, err := m.User.Search(query,
			management.Context(ctx),
			management.Page(page))
		if err != nil {
			return err
		}
		if !allowMultiple && (len(res.Users) > 1 || res.HasNext()) {
			return errors.New("multiple users returned for search")
		}
		for _, user := range res.Users {
			if err := fn(user); err != nil {
				return err
			}
		}
		if !res.HasNext() {
			break
		}
		page++
	}
	return nil
}

type userOutputFields struct {
	Email    string `json:"email"`
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
}

func userOutput(u *management.User) userOutputFields {
	var email string
	if u.Email != nil {
		email = *u.Email
	}
	var userID string
	if s, ok := u.AppMetadata["brim_user_id"].(string); ok {
		userID = s
	}
	var tenantID string
	if s, ok := u.AppMetadata["brim_tenant_id"].(string); ok {
		tenantID = s
	}
	return userOutputFields{
		Email:    email,
		UserID:   userID,
		TenantID: tenantID,
	}
}

func triggerChangePassword(ctx context.Context, a0cfg auth0ClientConfig, email string) error {
	type changePassword struct {
		Email      string `json:"email"`
		ClientID   string `json:"client_id"`
		Connection string `json:"connection"`
	}
	resp, err := resty.New().R().
		SetContext(ctx).
		SetBody(changePassword{
			Email:      email,
			ClientID:   a0cfg.clientId,
			Connection: a0cfg.connection,
		}).
		Post(a0cfg.domain.String() + "/dbconnections/change_password")
	if err != nil {
		return err
	}
	if !resp.IsSuccess() {
		return fmt.Errorf("failed to issue change password request: %v %v", resp.StatusCode(), string(resp.Body()))
	}
	return nil
}

func newTenantID() string {
	return "tenant_" + ksuid.New().String()
}

func validTenantID(s string) error {
	if !strings.HasPrefix(s, "tenant_") || len(s) != 34 {
		return errors.New("tenant id must start with \"tenant_\" and be 34 characters long")
	}
	return nil
}

func newUserID() string {
	return "user_" + ksuid.New().String()
}

func newPassword() string {
	buf := make([]byte, 16)
	_, err := rand.Read(buf)
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(buf)
}

var User = &charm.Spec{
	Name:  "user",
	Usage: "zident [global options] user command [options] [arguments...]",
	Short: "Create or edit users for a zqd service.",
	Long: `
zident user commands are used to create or edit user information in an Auth0 
tenant that will be used by a zqd service. The following environment variables
must be set with correct values for the Auth0 tenant and client in order to 
authenticate with the Auth0 management API: AUTH0_CLIENT_ID,
AUTH0_CLIENT_SECRET, AUTH0_DOMAIN, and AUTH0_CONNECTION.
`,
	New: New,
}

type Command struct {
	*root.Command
}

func init() {
	User.Add(NewUser)
	User.Add(ResetPassword)
	User.Add(Search)
	root.Zident.Add(User)
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	return c, nil
}

func (c *Command) Run(args []string) error {
	if len(args) == 0 {
		return root.Zident.Exec(c, []string{"help", "user"})
	}
	return charm.ErrNoRun
}
