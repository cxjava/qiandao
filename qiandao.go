package main

import (
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"
	"unicode/utf8"

	log "github.com/cihub/seelog"
	"github.com/robfig/cron"
)

var (
	dm = make(map[string]*Domain)
)

type Domain struct {
	ReqURL  string            `json:"url"`
	Method  string            `json:"method"`
	Params  map[string]string `json:"params"`
	Headers map[string]string `json:"headers"`
	Cookies map[string]string `json:"cookies"`
}

func (d *Domain) DoRequest() (*http.Response, error) {
	//add params
	val := url.Values{}
	if len(d.Params) > 0 {
		for k, v := range d.Params {
			val.Add(k, v)
		}
	}
	var body io.Reader
	u := d.ReqURL
	if d.Method == "GET" {
		body = nil
		if len(d.Params) > 0 {
			if strings.Contains(u, `?`) {
				u += `&`
			} else {
				u += `?`
			}
			u += val.Encode()
		}
	} else {
		body = strings.NewReader(val.Encode())
	}
	request, err := http.NewRequest(d.Method, u, body)
	if err != nil {
		log.Error(d.ReqURL, " http.NewRequest error:", err)
		return nil, err
	}
	//add header
	if len(d.Headers) > 0 {
		for k, v := range d.Headers {
			request.Header.Set(k, v)
		}
	}
	if len(d.Cookies) > 0 {
		for k, v := range d.Cookies {
			cookie := &http.Cookie{
				Name:     k,
				Value:    v,
				Expires:  time.Now().Add(356 * 24 * time.Hour),
				HttpOnly: true,
			}
			request.AddCookie(cookie)
		}
	}

	return httpClient.Do(request)
}

//读取响应
func ParseResponseBody(resp *http.Response) string {
	var body []byte
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			log.Error(err)
			return ""
		}
		defer reader.Close()
		bodyByte, err := ioutil.ReadAll(reader)
		if err != nil {
			log.Error(err)
			return ""
		}
		body = bodyByte
	default:
		bodyByte, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Error(err)
			return ""
		}
		body = bodyByte
	}
	return string(body)
}

func (d *Domain) getContent() (string, error) {
	content := ""
	resp, err := d.DoRequest()
	if err != nil {
		log.Error(d.ReqURL, " DoRequest error:", err)
		return content, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return ParseResponseBody(resp), nil
	} else if resp.StatusCode == 521 {
		return ParseResponseBody(resp), nil
	}
	log.Error(d.ReqURL, resp.StatusCode, ParseResponseBody(resp))
	return content, fmt.Errorf("StatusCode is not 200,", resp.StatusCode)
}

func smzdm() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	time.Sleep(time.Duration(r.Intn(40)) * time.Minute)
	time.Sleep(time.Duration(r.Intn(400)) * time.Second)
	content, err := dm["smzdm_home"].getContent()
	if err != nil {
		log.Error("smzdm", err)
		return
	}
	cookieTemp := SubString(content, `[];c.push("`, `");c=c.join("")`)
	cookieTemp = strings.Replace(cookieTemp, `");c.push("`, "", -1)
	dm["smzdm_user_info"].Headers["Cookie"] = `__jsl_clearance=` + cookieTemp
	dm["smzdm_login"].Headers["Cookie"] = `__jsl_clearance=` + cookieTemp
	dm["smzdm_qiandao"].Headers["Cookie"] = `__jsl_clearance=` + cookieTemp

	content, err = dm["smzdm_login"].getContent()
	if err != nil {
		log.Error("smzdm", err)
		return
	}
	if !strings.Contains(content, `"error_code":0,`) {
		log.Error("login failed！", content)
		return
	}
	log.Info("login success！", content)

	content, err = dm["smzdm_user_info"].getContent()
	if err != nil {
		log.Error("smzdm", err)
		return
	}
	log.Info("user_info:", content)

	content, err = dm["smzdm_qiandao"].getContent()
	if err != nil {
		log.Error("smzdm", err)
		return
	}
	if strings.Contains(content, `"error_code":0,`) {
		log.Info("qiandao success！", content)
	} else {
		log.Error("qiandao failed！", content)
	}
}

func kjl() {
	log.Info("kujiale!")
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	time.Sleep(time.Duration(r.Intn(18)) * time.Minute)
	time.Sleep(time.Duration(r.Intn(100)) * time.Second)
	content, err := dm["kujiale_main"].getContent()
	if err != nil {
		log.Error("kjl", err)
		return
	}
	content, err = dm["kujiale_login"].getContent()
	if err != nil {
		log.Error("kjl", err)
		return
	}
	log.Debug("login success！", content)
	content, err = dm["kujiale_qiandao"].getContent()
	if err != nil {
		log.Error("kjl", err)
		return
	}
	log.Info("kjl qiandao success！", content)
}

func MyCron() {
	c := cron.New()
	//9点1分11秒
	c.AddFunc(Conf.SmzdmCron, func() { go smzdm() })
	// c.AddFunc(Conf.KjlCron, func() { go kjl() })
	c.Start()
}

func init() {
	flag.Parse()
	ReadConfig()
	InitProxy()
}

func main() {
	defer log.Flush()
	defer func() {
		if err := recover(); err != nil {
			log.Critical(err)
			log.Critical(string(debug.Stack()))
		}
	}()
	log.Info("Start!!")
	readReq()
	MyCron()
	log.Info("Listen port 8000.")
	http.ListenAndServe(":8000", nil)
}

func readReq() {
	file, err := ioutil.ReadFile("./req.json")
	if err != nil {
		log.Error("File error: %v\n", err)
	}

	err = json.Unmarshal(file, &dm)
	if err != nil {
		log.Error("File error: %v\n", err)
	}
}

func SubString(str, begin, end string) (substr string) {
	// 将字符串的转换成[]rune
	rs := []rune(str)
	lth := len(rs)
	//开始位置获取
	beginIndex := UnicodeIndex(str, begin) + len([]rune(begin))
	if beginIndex < 0 {
		beginIndex = 0
	}
	if beginIndex >= lth {
		beginIndex = lth
	}
	// 结束位置获取
	endIndex := beginIndex + UnicodeIndex(string(rs[beginIndex:]), end)
	if endIndex < 0 {
		endIndex = 0
	}
	if endIndex >= lth {
		endIndex = lth
	}
	// 返回子串
	return string(rs[beginIndex:endIndex])
}

func UnicodeIndex(str, substr string) int {
	// 子串在字符串的字节位置
	result := strings.Index(str, substr)
	return utf8.RuneCountInString(str[:result])
}
