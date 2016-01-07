package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hybridgroup/gobot"
	"github.com/hybridgroup/gobot/platforms/i2c"
	"github.com/hybridgroup/gobot/platforms/raspi"
	"github.com/spf13/viper"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

var rgb string
var lcd_message string

type StatsdJson []struct {
	Datapoints [][]interface{} `json:"datapoints"`
	Target     string          `json:"target"`
}

type Counter struct {
	ResponseCode int
	Count        int
	RespTime     float64
}

func main() {
	if SetViper() {
		gbot := gobot.NewGobot()
		board := raspi.NewRaspiAdaptor("raspi")
		screen := i2c.NewGroveLcdDriver(board, "screen")
		// rgb_tmp := ""
		for {
			work := func() {

				if lcd_message == "" {
					fmt.Println("Loading...")
					screen.Write("Loading...")
				} else if lcd_message != "" {
					s := lcd_message[0:16] + "\n" + lcd_message[16:len(lcd_message)]
					fmt.Println(s)
					screen.Write(s)
				}
				// if rgb_tmp != rgb {
				if rgb == "green" {
					screen.SetRGB(0, 255, 0)
				}
				if rgb == "amber" {
					screen.SetRGB(255, 102, 0)
				}
				if rgb == "red" {
					screen.SetRGB(255, 0, 0)
				}
				// rgb_tmp = rgb
				// }

			}
			robot := gobot.NewRobot("screenBot",
				[]gobot.Connection{board},
				[]gobot.Device{screen},
				work,
			)
			gbot.AddRobot(robot)
			robot.Start()
			Looper()
			time.Sleep(10 * time.Second)
		}
	}
}

func SetViper() bool {
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	viper.SetConfigName("dev")
	viper.AddConfigPath("config")
	viper.AddConfigPath(dir + "/config/")
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Printf("%v cannot set viper: %v \n", time.Now(), err.Error())
		fmt.Printf("path:%v \n", dir)
		return false
	}
	return true
}

func Looper() {

	start := time.Now()
	codes := []int{200, 400, 401, 404, 500, 503}
	env := []string{"prod1", "prod2"}
	counter := make(chan Counter)
	for _, code := range codes {
		go curlStatsD(code, counter, env)
	}
	rgb = "green"
	lcd_message = ""
	m := make(map[int]string)

	for range codes {
		counter_back := <-counter
		fmt.Printf("%.2fs: %v - %v\n", counter_back.RespTime, counter_back.ResponseCode, counter_back.Count)
		if counter_back.ResponseCode == 200 {
			m[200] = strconv.Itoa(counter_back.Count)
		}
		if counter_back.ResponseCode == 400 {
			m[400] = strconv.Itoa(counter_back.Count)
		}
		if counter_back.ResponseCode == 401 {
			m[401] = strconv.Itoa(counter_back.Count)
		}
		if counter_back.ResponseCode == 404 {
			m[404] = strconv.Itoa(counter_back.Count)
		}
		if counter_back.ResponseCode == 503 {
			m[503] = strconv.Itoa(counter_back.Count)
		}
		if counter_back.ResponseCode == 500 {
			m[500] = strconv.Itoa(counter_back.Count)
		}
	}

	m503, _ := strconv.Atoi(m[503])
	m404, _ := strconv.Atoi(m[404])
	m401, _ := strconv.Atoi(m[401])
	m400, _ := strconv.Atoi(m[400])
	m500, _ := strconv.Atoi(m[500])
	if m503 > 0 || m404 > 0 || m401 > 0 || m400 > 0 {
		rgb = "amber"
	}
	if m500 > 0 {
		rgb = "red"
	}

	var keys []int
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		lcd_message += strconv.Itoa(k) + ":" + m[k] + " "
	}
	fmt.Printf("%.2fs elapsed\n", time.Since(start).Seconds())
	fmt.Printf("---------------------\n")
}

func curlStatsD(resp_code int, counter_chan chan Counter, envs []string) {
	start := time.Now()
	data := url.Values{}
	now := time.Now()
	then := now.Add(-10 * time.Minute)
	date := then.Format("20060102")
	t := then.Format("15:04")
	env_count := 0
	un := viper.GetString("graphite.un")
	pw := viper.GetString("graphite.pw")
	if un == "" && pw == "" {
		fmt.Println("cannot retrieve viper")
	} else {
		for _, env := range envs {
			x := fmt.Sprintf("https://"+un+":"+pw+"@graphite.mediamath.com/render?target=infra.apps_api.pops.prod."+env+".resp_code.%v.count&from=%v_%v&format=json", strconv.Itoa(resp_code), t, date)
			u, _ := url.ParseRequestURI(x)
			urlStr := fmt.Sprintf("%v", u)
			client := &http.Client{}
			r, _ := http.NewRequest("GET", urlStr, bytes.NewBufferString(data.Encode()))
			r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
			resp, do_err := client.Do(r)
			if do_err != nil {
				fmt.Println(do_err.Error())
				break
			} else {
				counter := 0
				defer resp.Body.Close()
				body, _ := ioutil.ReadAll(resp.Body)
				s := StatsdJson{}
				json.Unmarshal(body, &s)
				if len(s) > 0 {
					for _, point := range s[0].Datapoints {
						if point[0] != nil {
							count, ok := point[0].(float64)
							if ok {
								counter += int(count)
							}
						}
					}
				}
				env_count += counter
			}
		}
	}
	counter_chan <- Counter{resp_code, env_count, time.Since(start).Seconds()}
}
