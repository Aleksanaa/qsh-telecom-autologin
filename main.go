package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// So what's this and why is it there?
var confusingString = ">111111111"

var baseHeader = map[string]string{
	"Accept":          "*/*",
	"Accept-Encoding": "gzip, deflate",
	"Accept-Language": "en,zh-CN;q=0.7",
	"Connection":      "keep-alive",
	"Content-Type":    "application/x-www-form-urlencoded; charset=UTF-8",
	"User-Agent":      "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/118.0",
}

type loginClient struct {
	c           http.Client
	localIP     string
	cachePath   string
	username    string
	password    string
	exponent    string
	modulus     string
	passwordEnc string
	initHost    string
	loginHost   string
	queryString string
	userIndex   string
}

func (c *loginClient) Get(urlString string) *http.Response {
	req, err := http.NewRequest("GET", urlString, nil)
	if err != nil {
		log.Panic("Cannot make request: ", err)
	}

	return c.Do(req)
}

func (c *loginClient) Post(urlString string, body io.Reader) *http.Response {
	req, err := http.NewRequest("POST", urlString, body)
	if err != nil {
		log.Panic("Cannot make request: ", err)
	}

	return c.Do(req)
}

func (c *loginClient) Do(req *http.Request) *http.Response {
	var dialContext func(ctx context.Context, network, addr string) (net.Conn, error)

	if c.localIP != "" {
		localIP := net.ParseIP(c.localIP)
		if localIP == nil {
			log.Fatalf("Invalid local IP: %s", c.localIP)
		}

		dialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			localAddr := &net.TCPAddr{
				IP: localIP,
			}
			d := net.Dialer{
				LocalAddr: localAddr,
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}
			return d.DialContext(ctx, network, addr)
		}
	} else {
		dialer := &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}
		dialContext = dialer.DialContext
	}

	c.c.Transport = &http.Transport{
		DialContext: dialContext,
	}

	for k, v := range baseHeader {
		req.Header.Add(k, v)
	}
	// disable 302 redirect in http module itself
	c.c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	c.c.Timeout = 20 * time.Second

	resp, err := c.c.Do(req)
	if err != nil {
		log.Panic("Cannot connect: ", err)
	}

	return resp
}

func (c *loginClient) PasswordEncrypt() {
	if (c.modulus != "") && (c.exponent != "") && (c.password != "") {
		c.password = c.password + confusingString
		// just very simple RSA with no padding
		m, _ := new(big.Int).SetString(c.modulus, 16)
		e, _ := new(big.Int).SetString(c.exponent, 16)
		p := new(big.Int).SetBytes([]byte(c.password))
		crypted := new(big.Int).Exp(p, e, m)
		c.passwordEnc = hex.EncodeToString(crypted.Bytes())
	} else if c.passwordEnc != "" {
		return
	} else if c.password == "" {
		log.Panic("Cannot encrypt password: password not given")
	} else {
		log.Panic("Cannot encrypt password: not enough arguments")
	}
}

func (c *loginClient) myPost(urlString string, reqData map[string]string, respData interface{}) {
	formData := url.Values{}
	for key, value := range reqData {
		formData.Add(key, value)
	}
	body := strings.NewReader(formData.Encode())
	resp := c.Post(urlString, body)
	_ = json.NewDecoder(resp.Body).Decode(respData)
	defer resp.Body.Close()
}

func (c *loginClient) loginInit() {
	urlString := "http://" + c.initHost
	for {
		resp := c.Get(urlString)
		if resp.StatusCode == http.StatusFound {
			urlString = resp.Header.Get("Location")
		} else {
			u, err := url.Parse(urlString)
			if err != nil {
				log.Panic("Returned illegal url '", urlString, "': ", err)
			}
			c.loginHost = u.Host
			c.queryString = u.RawQuery
			break
		}
	}
}

func (c *loginClient) getEncryptKey() {
	urlString := "http://" + c.loginHost + "/eportal/InterFace.do?method=pageInfo"
	reqData := map[string]string{
		"queryString": c.queryString,
	}
	type respStruct struct {
		PublicKeyExponent string
		PublicKeyModulus  string
	}
	respData := respStruct{}
	c.myPost(urlString, reqData, &respData)
	c.exponent = respData.PublicKeyExponent
	if c.modulus != respData.PublicKeyModulus {
		if c.modulus != "" {
			log.Print("Encryption modulus is changed")
		}
		c.modulus = respData.PublicKeyModulus
	}
	c.PasswordEncrypt()
}

func (c *loginClient) login() {
	urlString := "http://" + c.loginHost + "/eportal/InterFace.do?method=login"
	reqData := map[string]string{
		"userId":          c.username,
		"password":        c.passwordEnc,
		"service":         "",
		"queryString":     c.queryString,
		"operatorPwd":     "",
		"operatorUserId":  "",
		"validcode":       "",
		"passwordEncrypt": "true",
	}
	type respStruct struct {
		Result    string
		UserIndex string
	}
	respData := respStruct{}
	c.myPost(urlString, reqData, &respData)
	if respData.Result == "success" {
		c.userIndex = respData.UserIndex
		log.Print("Successfully logged in with account '", c.username, "'")
	} else {
		log.Panic("Login attempt failed with account '", c.username, "'")
	}
}

func (c *loginClient) logout() {
	urlString := "http://" + c.initHost + "/eportal/InterFace.do?method=logout"
	reqData := map[string]string{
		"userIndex": c.userIndex,
	}
	type respStruct struct {
		Result string
	}
	respData := respStruct{}
	c.myPost(urlString, reqData, &respData)
	if respData.Result == "success" {
		log.Print("Successfully logged out")
	} else {
		log.Panic("Logout attempt failed, maybe user index has expired")
	}
}

type cache struct {
	Username    string
	PasswordEnc string
	InitHost    string
	UserIndex   string
	Modulus     string
}

func (c *loginClient) loadCache() {
	if c.cachePath == "" {
		return
	}
	path, _ := filepath.Abs(c.cachePath)
	file, err := os.ReadFile(path)
	if err != nil {
		return
	}
	fileCache := cache{}
	err = json.Unmarshal([]byte(file), &fileCache)
	if err != nil {
		log.Panic("Cannot parse cache file: ", err)
	}
	if c.username == "" {
		c.username = fileCache.Username
	}
	if c.initHost == "" {
		c.initHost = fileCache.InitHost
	}
	c.passwordEnc = fileCache.PasswordEnc
	if c.userIndex == "" {
		c.userIndex = fileCache.UserIndex
	}
	c.modulus = fileCache.Modulus
}

func (c *loginClient) saveCache() {
	if c.cachePath == "" {
		return
	}
	fileCache := cache{
		Username:    c.username,
		PasswordEnc: c.passwordEnc,
		InitHost:    c.initHost,
		UserIndex:   c.userIndex,
		Modulus:     c.modulus,
	}
	file, _ := json.MarshalIndent(fileCache, "", " ")
	path, _ := filepath.Abs(c.cachePath)
	err := os.WriteFile(path, file, 0666)
	if err != nil {
		log.Panic("Failed to write to cache file: ", err)
	}
}

func (c *loginClient) run() {
	flag.StringVar(&c.username, "name", "", "Account name, usually phone number")
	flag.StringVar(&c.password, "passwd", "", "Password to the account")
	flag.StringVar(&c.initHost, "host", "172.25.249.70", "Domain of the login page, usually ip address")
	flag.StringVar(&c.cachePath, "cache", "", "Specify where to read and store cache, blank to disable")
	flag.StringVar(&c.userIndex, "index", "", "User Index of user, only for logging out")
	flag.StringVar(&c.localIP, "localip", "", "Local IP address to bind to")
	logout := flag.Bool("logout", false, "Whether to log out current user")
	flag.Parse()

	if !*logout {
		if (c.cachePath == "") && (c.username == "" || c.password == "") {
			log.Panic("Not enough argument for login. See --help for explanation")
		}
		c.loadCache()
		c.loginInit()
		c.getEncryptKey()
		c.login()
		c.saveCache()
	} else {
		if (c.cachePath == "") && (c.userIndex == "") {
			log.Panic("Not enough argument for logout. See --help for explanation")
		}
		c.loadCache()
		c.logout()
	}
}

func main() {
	defer os.Exit(1)
	client := &loginClient{}
	client.run()
	os.Exit(0)
}
