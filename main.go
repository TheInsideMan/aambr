package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hybridgroup/gobot"
	"github.com/hybridgroup/gobot/platforms/gpio"
	"github.com/hybridgroup/gobot/platforms/raspi"
	"github.com/spf13/viper"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type StatsdJson []struct {
	Datapoints [][]interface{} `json:"datapoints"`
	Target     string          `json:"target"`
}

func main() {
	if SetViper() {
		for {
			Looper()
			time.Sleep(10 * time.Second)
		}
	}
}

func BotWork(led *gpio.LedDriver) int {
	// gobot.Every(1*time.Second, func() {
	// 	err := led.Off()
	// 	if err != nil {
	// 		fmt.Println(err.Error())
	// 	}
	// 	time.Sleep(900 * time.Millisecond)
	// 	led.On()
	// })
	// for {
	led.On()
	time.Sleep(100 * time.Millisecond)
	led.Off()
	return 0
}

func SetViper() bool {
	viper.SetConfigName("dev")
	viper.AddConfigPath("config")
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println("cannot set viper")
		return false
	}
	return true
}

func Looper() {

	start := time.Now()
	codes := []int{200, 401, 404, 500, 503}
	env := []string{"prod1", "prod2"}
	counter := make(chan Counter)
	for _, code := range codes {
		go curlStatsD(code, counter, env)
	}

	// SET ROBOT
	gbot := gobot.NewGobot()
	r := raspi.NewRaspiAdaptor("raspi")
	green_led := gpio.NewLedDriver(r, "led", "12")
	// END SET ROBOT

	for range codes {
		counter_back := <-counter
		fmt.Printf("%.2fs: %v - %v\n", counter_back.RespTime, counter_back.ResponseCode, counter_back.Count)
		if counter_back.ResponseCode == 200 {
			if int(counter_back.Count) > 0 {
				green_led_robot := gobot.NewRobot("green_led_robot",
					[]gobot.Connection{r},
					[]gobot.Device{green_led},
					BotWork(green_led),
				)
				gbot.AddRobot(green_led_robot)
				green_led_robot.Start()
				// gbot.Start()
			}
		}
	}
	gbot.Stop()
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
			// fmt.Println(x)
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
							// _, ok := point[1].(float64)
							// if ok {
							// ni := int(n)
							// ns := strconv.Itoa(ni)
							// i, err := strconv.ParseInt(ns, 10, 64)
							// if err != nil {
							// 	panic(err)
							// }
							// tm := time.Unix(i, 0)
							// fmt.Println(tm)
							// }
						}
					}
				}
				env_count += counter
			}
		}
	}
	counter_chan <- Counter{resp_code, env_count, time.Since(start).Seconds()}
	// resp.Body.Close()
}

type Counter struct {
	ResponseCode int
	Count        int
	RespTime     float64
}
