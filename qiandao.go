package main

import (
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

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
}

func (d *Domain) DoResponse() (*http.Response, error) {
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
			u = u + "?" + val.Encode()
		}
	} else {
		body = strings.NewReader(val.Encode())
	}
	request, err := http.NewRequest(d.Method, u, body)
	if err != nil {
		fmt.Println(d.ReqURL, " http.NewRequest error:", err)
		Error(d.ReqURL, " http.NewRequest error:", err)
		return nil, err
	}
	//add header
	if len(d.Headers) > 0 {
		for k, v := range d.Headers {
			request.Header.Add(k, v)
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
			fmt.Println(err)
			return ""
		}
		defer reader.Close()
		bodyByte, err := ioutil.ReadAll(reader)
		if err != nil {
			fmt.Println(err)
			return ""
		}
		body = bodyByte
	default:
		bodyByte, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			return ""
		}
		body = bodyByte
	}
	return string(body)
}

func (d *Domain) getContent() (string, error) {
	content := ""
	resp, err := d.DoResponse()
	if err != nil {
		fmt.Println(d.ReqURL, " DoResponse error:", err)
		Error(d.ReqURL, " DoResponse error:", err)
		return content, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return ParseResponseBody(resp), nil
	}
	fmt.Println(d.ReqURL, resp.StatusCode, resp.Header, resp.Cookies())
	return content, fmt.Errorf("StatusCode is not 200", resp.StatusCode)
}

func smzdm() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	time.Sleep(time.Duration(r.Intn(40)) * time.Minute)
	time.Sleep(time.Duration(r.Intn(400)) * time.Second)
	content, err := dm["smzdm_home"].getContent()
	if err != nil {
		fmt.Println("smzdm", err)
		return
	}
	content, err = dm["smzdm_user_info"].getContent()
	if err != nil {
		fmt.Println("smzdm", err)
		return
	}
	content, err = dm["smzdm_login"].getContent()
	if err != nil {
		fmt.Println("smzdm", err)
		return
	}
	if !strings.Contains(content, `"error_code":0,`) {
		fmt.Println("登录失败！", content)
		return
	}
	fmt.Println("登录成功！", content)
	content, err = dm["smzdm_qiandao"].getContent()
	if err != nil {
		fmt.Println("smzdm", err)
		return
	}
	if strings.Contains(content, `"error_code":0,`) {
		fmt.Println("签到成功！", content)
	} else {
		fmt.Println("签到失败！", content)
	}
}

func kjl() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	time.Sleep(time.Duration(r.Intn(40)) * time.Minute)
	time.Sleep(time.Duration(r.Intn(400)) * time.Second)
	content, err := dm["kujiale_main"].getContent()
	if err != nil {
		fmt.Println("kjl", err)
		return
	}
	content, err = dm["kujiale_login"].getContent()
	if err != nil {
		fmt.Println("kjl", err)
		return
	}
	fmt.Println("登录成功！", content)
	content, err = dm["kujiale_qiandao"].getContent()
	if err != nil {
		fmt.Println("kjl", err)
		return
	}
	fmt.Println("签到成功！", content)
}

func MyCron() {
	c := cron.New()
	//9点1分11秒
	c.AddFunc(Conf.SmzdmCron, func() { go smzdm() })
	c.AddFunc(Conf.KjlCron, func() { go kjl() })
	c.Start()
}

func init() {
	flag.Parse()
	ReadConfig()
	SetLogInfo()
	InitProxy()
}

func main() {
	readReq()
	MyCron()
	// smzdm()
	// kjl()
	Info("Listen port 8000.")
	http.ListenAndServe(":8000", nil)
}

func readReq() {
	file, err := ioutil.ReadFile("./req.json")
	if err != nil {
		fmt.Printf("File error: %v\n", err)
	}

	err = json.Unmarshal(file, &dm)
	if err != nil {
		fmt.Printf("File error: %v\n", err)
	}
}
func SetLogInfo() {
	// debug 1, info 2
	SetLevel(2)
	SetLogger("console", "")
	SetLogger("file", `{"filename":"log.log","daily":false}`)
}
