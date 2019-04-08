package rancher

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"

	"github.com/vmware/harbor/src/common/dao"
	"github.com/vmware/harbor/src/common/models"
	"github.com/vmware/harbor/src/common/utils/log"
	"github.com/vmware/harbor/src/ui/auth"
)

type Auth struct {
	auth.DefaultAuthenticateHelper
}

type Token struct {
	Token          string `json:"token" norman:"writeOnly,noupdate"`
	UserPrincipal  string `json:"userPrincipal" norman:"type=reference[principal]"`
	UserID         string `json:"userId" norman:"type=reference[user]"`
	AuthProvider   string `json:"authProvider"`
	TTLMillis      int64  `json:"ttl"`
	LastUpdateTime string `json:"lastUpdateTime"`
	IsDerived      bool   `json:"isDerived"`
	Description    string `json:"description"`
	Expired        bool   `json:"expired"`
	ExpiresAt      string `json:"expiresAt"`
	Current        bool   `json:"current"`
}

const tokenPattern = `token-[a-z0-9]{5}:[a-z0-9]{54}`

// Authenticate calls dao to authenticate user.
func (d *Auth) Authenticate(m models.AuthModel) (*models.User, error) {
	if d.isToken(m) {
		return d.authByRancherToken(m.Principal, m.Password)
	} else {
		normalUrl := rancherLoginURL()
		localUrl := rancherLocalLoginURL()
		user, err := d.authByBasicAuth(normalUrl, m.Principal, m.Password)
		if err != nil && localUrl != "" && normalUrl != localUrl {
			user, err = d.authByBasicAuth(localUrl, m.Principal, m.Password)
		}
		return user, err
	}
}

func (d *Auth) isToken(m models.AuthModel) bool {
	if len(m.Password) == 66 {
		re := regexp.MustCompile(tokenPattern)
		s := re.FindStringSubmatch(m.Password)
		if len(s) == 1 {
			return true
		}
	}
	return false
}

func (d *Auth) authByBasicAuth(url, username, password string) (*models.User, error) {
	rLogin := &rancherLogin{
		Username:     username,
		Password:     password,
		ResponseType: "session",
		TTLMillis:    3000000,
	}
	bytesData, err := json.Marshal(rLogin)
	if err != nil {
		return nil, err
	}
	response, err := httpClient.Post(url, "application/json", bytes.NewReader(bytesData))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	u := models.User{}
	if response.StatusCode == http.StatusCreated {
		u.Username = rLogin.Username
		u.Realname = rLogin.Username
		u.Password = password

		b, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}
		var token Token
		err = json.Unmarshal(b, &token)
		if err != nil {
			return nil, err
		}
		if len(token.Token) == 0 {
			return nil, auth.NewErrAuth("Invalid credentials")
		}
	} else {
		return nil, auth.NewErrAuth("Invalid credentials")
	}
	return &u, nil
}

type rancherUserInfo struct {
	Data []rancherUserData `json:"data"`
}

type rancherUserData struct {
	Id       string `json:"id"`
	Enabled  bool   `json:"enabled"`
	Name     string `json:"name"`
	Username string `json:"username"`
}

func (d *Auth) authByRancherToken(username, principal string) (*models.User, error) {
	request, err := http.NewRequest(http.MethodGet, rancherUserInfoURL(), nil)
	if err != nil {
		return nil, err
	}
	c := &http.Cookie{}
	c.Path = "/"
	c.HttpOnly = true
	c.Secure = true
	c.Name = "R_SESS"
	c.Value = principal
	request.AddCookie(c)

	response, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, auth.NewErrAuth("Invalid credentials")
	}
	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	var userInfo rancherUserInfo
	err = json.Unmarshal(b, &userInfo)
	if err != nil {
		return nil, err
	}
	for _, d := range userInfo.Data {
		if d.Enabled && (d.Username == username || d.Username == "admin") {
			u := models.User{
				Username: username,
				Realname: d.Name,
				Password: principal,
			}
			return &u, nil
		}
	}

	return nil, auth.NewErrAuth("Invalid credentials")
}

func (u *Auth) PostAuthenticate(user *models.User) error {
	dbUser, err := dao.GetUser(models.User{Username: user.Username})
	if err != nil {
		return err
	}
	if dbUser == nil {
		return u.OnBoardUser(user)
	}
	user.UserID = dbUser.UserID
	user.HasAdminRole = dbUser.HasAdminRole
	fillEmailRealName(user)
	if err2 := dao.ChangeUserProfile(*user, "Email", "Realname"); err2 != nil {
		log.Warningf("Failed to update user profile, user: %s, error: %v", user.Username, err2)
	}

	return nil
}

// SearchUser - Check if user exist in local db
func (d *Auth) SearchUser(username string) (*models.User, error) {
	var queryCondition = models.User{
		Username: username,
	}

	return dao.GetUser(queryCondition)
}

// OnBoardUser -
func (d *Auth) OnBoardUser(u *models.User) error {
	fillEmailRealName(u)
	u.Password = "12345678AbC"  //Password is not kept in local db
	u.Comment = "from RANCHER." //Source is from RANCHER

	return dao.OnBoardUser(u)
}

func fillEmailRealName(user *models.User) {
	if len(user.Realname) == 0 {
		user.Realname = user.Username
	}
	if len(user.Email) == 0 {
		user.Email = user.Username + "@rancher.placeholder"
	}
}

var httpClient *http.Client

func init() {
	auth.Register("rancher_auth", &Auth{})
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient = &http.Client{Transport: tr}
}

func rancherUserInfoURL() string {
	return os.Getenv("RANCHER_USER_INFO_URL")
}

func rancherLoginURL() string {
	return os.Getenv("RANCHER_LOGIN_URL")
}

func rancherLocalLoginURL() string {
	return os.Getenv("RANCHER_LOCAL_LOGIN_URL")
}

type rancherLogin struct {
	Username     string `json:"username" norman:"type=string,required"`
	Password     string `json:"password" norman:"type=string,required"`
	ResponseType string `json:"responseType,omitempty" norman:"type=string,required"`
	TTLMillis    int64  `json:"ttl,omitempty"`
}
