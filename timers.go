package main

import (
	"encoding/json"
	"io/ioutil"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

const STOP_AFTER = 10 //seconds
type rule struct {
	Frequency string
	Days      []string
	Start     string
	End       string
}

func start() {
	log.Println("Starting timer thread")

	var alarms map[string]map[int][]int

	//check if any timers have already been setup
	f := getOsFilePath(SETTINGS_FILE)
	settings, err := ioutil.ReadFile(f)

	if err == nil {
		rules := parseRules(string(settings))
		log.Println(rules)
		alarms = getAlarms(rules)
		log.Println(alarms)
	}

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case jsonstr := <-timerCh:
			//jsonstr = `[{"frequency":"1","days":["Tuesday"], "start": "09", "end": "10"}]`
			rules := parseRules(jsonstr)
			log.Println(rules)
			alarms = getAlarms(rules)
			log.Println(alarms)

		case t := <-ticker.C:
			log.Printf("Current time: %s %d %d", t.Weekday(), t.Hour(), t.Minute())
			for _, m := range alarms[t.Weekday().String()][t.Hour()] {
				if m == t.Minute() {
					playerCh <- PLAY
					log.Printf("Playing alarm")
					time.Sleep(STOP_AFTER * time.Second)
					playerCh <- STOP //stop alarm after STOP_AFTER unconditionally
				}
			}
		}
	}
}

//parse rules received in json format from UI
func parseRules(jsonstr string) []rule {
	var rules []rule
	err := json.Unmarshal([]byte(jsonstr), &rules)
	if err != nil {
		log.Println(err)
	}

	return rules
}

//for each day, for each hour find the minutes at which alarm should be sounded
func getAlarms(rules []rule) map[string]map[int][]int {
	var alarms = make(map[string]map[int][]int)
	for _, r := range rules {
		hours := make(map[int][]int)
		for _, d := range r.Days {
			alarms[d] = hours
			s, _ := strconv.Atoi(r.Start)
			e, _ := strconv.Atoi(r.End)
			f, _ := strconv.Atoi(r.Frequency)
			hrs := getHours(s, e)

			i := 0
			m := f
			h := 0

			for {
				mins := make([]int, 0)
				for {
					mins = append(mins, m%60)
					m += f

					if m-(60*i) >= 60 {

						h = hrs[i]
						if f == 60 && h == s {
							//if the frequency is 60 minutes, do not play on the starting hour
							i++
							break
						}

						alarms[d][h] = mins
						log.Printf("%s h: %v mins: %v\n", d, h, mins)
						i++
						break
					}
				}
				if e == hrs[i] {
					//if the alarm falls exactly on the end hour we should play it
					if m%60 == 0 {
						alarms[d][e] = []int{0}
						log.Printf("%s h: %v mins: %v\n", d, e, alarms[d][e])
					}
					break
				}
			}
		}
	}
	return alarms
}

//get all hours from start to end , both inclusive
func getHours(s, e int) []int {
	var hrs []int

	for {
		hrs = append(hrs, s)
		s += 1
		if s > e {
			break
		}
	}
	return hrs
}
